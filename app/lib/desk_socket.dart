import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:crypto/crypto.dart' as crypto;
import 'package:cryptography/cryptography.dart';
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/status.dart' as status;

enum DeskConnState { connecting, connected, disconnected, reconnecting, closed }

class DeskConnException implements Exception {
  final String code;
  final String message;
  DeskConnException(this.code, this.message);
  @override
  String toString() => "$code: $message";
}

class DeskSocket {
  final String url;

  final bool useTls;
  final String certFpSha256Hex; // hex SHA-256

  final bool securePayload; // legacy AES-GCM on top
  final String token;

  final bool useUserPass;
  final String username;
  final String password;

  IOWebSocketChannel? _channel;
  StreamSubscription? _sub;

  final _pending = <String, Completer<Map<String, dynamic>>>{};
  final _incoming = StreamController<Map<String, dynamic>>.broadcast();
  final _conn = StreamController<DeskConnState>.broadcast();

  int _seq = 0;
  bool _closedByUser = false;
  bool _reconnectScheduled = false;
  int _reconnectAttempt = 0;

  DeskConnState _state = DeskConnState.disconnected;

  // crypto (payload AES-GCM)
  final _aes = AesGcm.with256bits();
  final _rng = Random.secure();
  SecretKey? _key;

  final _ready = Completer<void>();
  Future<void> get ready => _ready.future;

  DeskSocket._(
    this.url, {
    required this.useTls,
    required this.certFpSha256Hex,
    required this.securePayload,
    required this.token,
    required this.useUserPass,
    required this.username,
    required this.password,
  }) {
    _open();
  }

  static DeskSocket connect(
    String url, {
    required bool useTls,
    required String certFpSha256Hex,
    required bool securePayload,
    required String token,
    required bool useUserPass,
    required String username,
    required String password,
  }) {
    return DeskSocket._(
      url,
      useTls: useTls,
      certFpSha256Hex: certFpSha256Hex,
      securePayload: securePayload,
      token: token,
      useUserPass: useUserPass,
      username: username,
      password: password,
    );
  }

  Stream<Map<String, dynamic>> get messages => _incoming.stream;
  Stream<DeskConnState> get connection => _conn.stream;
  DeskConnState get state => _state;

  void _setState(DeskConnState s) {
    _state = s;
    if (!_conn.isClosed) _conn.add(s);
  }

  Future<void> _initCryptoIfNeeded() async {
    if (!securePayload) {
      _key = null;
      return;
    }
    final t = token.trim();
    if (t.isEmpty) {
      _key = null;
      return;
    }
    final h = await Sha256().hash(utf8.encode(t));
    _key = SecretKey(h.bytes);
  }

  List<int> _nonce() => List<int>.generate(12, (_) => _rng.nextInt(256));

  Future<Map<String, dynamic>> _encryptEnvelope(Map<String, dynamic> plain) async {
    final k = _key;
    if (!securePayload || k == null) return plain;

    final nonce = _nonce();
    final aad = utf8.encode("deskcontrol-v1");
    final data = utf8.encode(jsonEncode(plain));

    final box = await _aes.encrypt(data, secretKey: k, nonce: nonce, aad: aad);

    return {
      "enc": 1,
      "nonce": base64Encode(nonce),
      "data": base64Encode(box.cipherText),
      "tag": base64Encode(box.mac.bytes),
    };
  }

  Future<Map<String, dynamic>> _decryptEnvelope(Map<String, dynamic> env) async {
    final k = _key;
    if (!securePayload || k == null) return env;

    final nonce = base64Decode((env["nonce"] ?? "").toString());
    final ct = base64Decode((env["data"] ?? "").toString());
    final tag = base64Decode((env["tag"] ?? "").toString());
    final aad = utf8.encode("deskcontrol-v1");

    final clear = await _aes.decrypt(
      SecretBox(ct, nonce: nonce, mac: Mac(tag)),
      secretKey: k,
      aad: aad,
    );

    final obj = jsonDecode(utf8.decode(clear));
    if (obj is! Map) throw DeskConnException("DECRYPT_INVALID", "Payload no es JSON map");
    return Map<String, dynamic>.from(obj);
  }

  static String _sha256Hex(List<int> der) {
    final d = crypto.sha256.convert(der);
    return d.bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  }

  HttpClient? _buildPinnedClientIfNeeded() {
    if (!useTls) return null;

    final fp = certFpSha256Hex.trim().toLowerCase();
    if (fp.isEmpty) return HttpClient();

    final client = HttpClient();
    client.badCertificateCallback = (X509Certificate cert, String host, int port) {
      final got = _sha256Hex(cert.der).toLowerCase();
      return got == fp;
    };
    return client;
  }

  /// ✅ Aquí está el fix: mandamos token en 2 headers + (la URL ya trae token en query)
  Map<String, dynamic>? _buildHeaders() {
    final headers = <String, dynamic>{};

    final t = token.trim();
    if (useTls && t.isNotEmpty) {
      // 1) header propio
      headers['X-DeskControl-Token'] = t;
      // 2) bearer (muchas implementaciones lo usan)
      headers['Authorization'] = 'Bearer $t';
    }

    // Basic Auth si está activado
    if (useUserPass && username.trim().isNotEmpty) {
      final basic = base64Encode(utf8.encode('${username.trim()}:$password'));
      headers['Authorization-Basic'] = 'Basic $basic';
      // Nota: no pisamos Authorization porque ya lo usamos para Bearer.
      // Si tu daemon usa Basic real, luego lo cambiamos a un solo esquema.
    }

    return headers.isEmpty ? null : headers;
  }

  Future<void> _open() async {
    if (_closedByUser) return;

    _setState(_reconnectAttempt == 0 ? DeskConnState.connecting : DeskConnState.reconnecting);

    try {
      await _initCryptoIfNeeded();

      if (useTls) {
        if (token.trim().isEmpty) {
          throw DeskConnException("TOKEN_REQUIRED", "Falta token (escanea el QR).");
        }
        if (certFpSha256Hex.trim().isEmpty) {
          throw DeskConnException("FP_REQUIRED", "Falta fingerprint (escanea el QR).");
        }
      }

      final client = _buildPinnedClientIfNeeded();
      final headers = _buildHeaders();

      final ws = await WebSocket.connect(
        url,
        headers: headers,
        compression: CompressionOptions.compressionDefault,
        customClient: client,
      );
      ws.pingInterval = const Duration(seconds: 20);

      _channel = IOWebSocketChannel(ws);

      _sub?.cancel();
      _sub = _channel!.stream.listen(
        (event) async {
          if (_state != DeskConnState.connected) {
            _reconnectAttempt = 0;
            _setState(DeskConnState.connected);
          }

          try {
            final obj = jsonDecode(event as String);
            if (obj is! Map) return;
            final raw = Map<String, dynamic>.from(obj);

            final msg = (raw["enc"] == 1) ? await _decryptEnvelope(raw) : raw;

            _incoming.add(msg);

            final id = msg['id']?.toString();
            if (id != null) {
              final c = _pending.remove(id);
              if (c != null && !c.isCompleted) c.complete(msg);
            }
          } catch (e) {
            _incoming.add({"type": "error", "error": e.toString()});
          }
        },
        onError: (e) {
          _failAllPending(e);
          _scheduleReconnect();
        },
        onDone: () {
          _failAllPending(StateError('socket closed'));
          _scheduleReconnect();
        },
        cancelOnError: true,
      );

      if (!_ready.isCompleted) {
        // ignore: discarded_futures
        _bootstrap();
      }
    } catch (e) {
      if (!_ready.isCompleted) {
        _ready.completeError(DeskConnException("WS_CONNECT_FAIL", e.toString()));
      }
      _failAllPending(e);
      _scheduleReconnect();
    }
  }

  Future<void> _bootstrap() async {
    Timer? timer;
    final waiter = Completer<Map<String, dynamic>>();
    StreamSubscription<Map<String, dynamic>>? sub;

    try {
      sub = messages.listen((m) {
        if (!waiter.isCompleted) waiter.complete(m);
      });

      timer = Timer(const Duration(seconds: 6), () {
        if (!waiter.isCompleted) {
          waiter.completeError(DeskConnException("HELLO_TIMEOUT", "No hubo respuesta inicial"));
        }
      });

      final helloId = '${DateTime.now().millisecondsSinceEpoch}_${_seq++}';
      await send({"id": helloId, "type": "ping", "app": "deskcontrol", "v": 1});

      await waiter.future;

      if (!_ready.isCompleted) _ready.complete();
    } catch (e) {
      if (!_ready.isCompleted) _ready.completeError(e);
      close();
    } finally {
      timer?.cancel();
      await sub?.cancel();
    }
  }

  void _failAllPending(Object e) {
    for (final c in _pending.values) {
      if (!c.isCompleted) c.completeError(e);
    }
    _pending.clear();
  }

  void _scheduleReconnect() {
    if (_closedByUser) return;
    if (_reconnectScheduled) return;
    _reconnectScheduled = true;

    _setState(DeskConnState.disconnected);
    _setState(DeskConnState.reconnecting);

    final delaySeconds = (_reconnectAttempt <= 0)
        ? 1
        : (_reconnectAttempt == 1)
            ? 2
            : (_reconnectAttempt == 2)
                ? 3
                : (_reconnectAttempt == 3)
                    ? 5
                    : 8;

    _reconnectAttempt++;

    Timer(Duration(seconds: delaySeconds), () {
      _reconnectScheduled = false;
      if (_closedByUser) return;
      _cleanupChannel();
      _open();
    });
  }

  void _cleanupChannel() {
    _sub?.cancel();
    _sub = null;

    try {
      _channel?.sink.close(status.normalClosure);
    } catch (_) {}
    _channel = null;
  }

  Future<void> send(Map<String, dynamic> msg) async {
    if (_closedByUser) return;
    if (_channel == null) return;

    final out = await _encryptEnvelope(msg);
    _channel!.sink.add(jsonEncode(out));
  }

  Future<Map<String, dynamic>> request(
    String type, {
    Map<String, dynamic>? payload,
    Duration timeout = const Duration(seconds: 10),
  }) {
    final id = '${DateTime.now().millisecondsSinceEpoch}_${_seq++}';
    final c = Completer<Map<String, dynamic>>();
    _pending[id] = c;

    final msg = <String, dynamic>{
      'id': id,
      'type': type,
      if (payload != null) ...payload,
    };

    // ignore: discarded_futures
    send(msg);

    return c.future.timeout(timeout, onTimeout: () {
      _pending.remove(id);
      throw DeskConnException("TIMEOUT", "Timeout esperando respuesta de $type");
    });
  }

  void close() {
    _closedByUser = true;
    _setState(DeskConnState.closed);

    _failAllPending(StateError('socket closed'));
    _cleanupChannel();

    _incoming.close();
    _conn.close();
  }
}
