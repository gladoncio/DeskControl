import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/status.dart' as status;

enum DeskConnState { connecting, connected, disconnected, reconnecting, closed }

class DeskSocket {
  final String url;

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

  DeskSocket._(this.url) {
    _open();
  }

  static DeskSocket connect(String url) {
    return DeskSocket._(url);
  }

  Stream<Map<String, dynamic>> get messages => _incoming.stream;
  Stream<DeskConnState> get connection => _conn.stream;
  DeskConnState get state => _state;

  bool get isConnected => _state == DeskConnState.connected;

  void _setState(DeskConnState s) {
    _state = s;
    if (!_conn.isClosed) _conn.add(s);
  }

  void _open() {
    if (_closedByUser) return;

    _setState(_reconnectAttempt == 0
        ? DeskConnState.connecting
        : DeskConnState.reconnecting);

    try {
      // ✅ Ping real de WebSocket (no mensajes JSON) -> evita que el router/PC corte por idle
      _channel = IOWebSocketChannel.connect(
        Uri.parse(url),
        pingInterval: const Duration(seconds: 20),
        connectTimeout: const Duration(seconds: 5),
      );

      _sub?.cancel();
      _sub = _channel!.stream.listen(
        (event) {
          // primer mensaje recibido => conectado
          if (_state != DeskConnState.connected) {
            _reconnectAttempt = 0;
            _setState(DeskConnState.connected);
          }

          try {
            final obj = jsonDecode(event as String);
            if (obj is! Map) return;
            final msg = Map<String, dynamic>.from(obj);

            _incoming.add(msg);

            final id = msg['id']?.toString();
            if (id != null) {
              final c = _pending.remove(id);
              if (c != null && !c.isCompleted) c.complete(msg);
            }
          } catch (_) {
            // ignore parse
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

      // si conecta pero no llega ningún msg igual queremos considerarlo conectado:
      // dejamos que el primer send también marque activity, pero el ping ya ayuda.
      // Marcamos "connecting" por ahora, pasará a connected al primer evento o al primer send.
    } catch (e) {
      _failAllPending(e);
      _scheduleReconnect();
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

    if (_state != DeskConnState.reconnecting) {
      _setState(DeskConnState.disconnected);
      _setState(DeskConnState.reconnecting);
    }

    // backoff simple
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
    try {
      _sub?.cancel();
    } catch (_) {}
    _sub = null;

    try {
      _channel?.sink.close(status.goingAway);
    } catch (_) {}
    _channel = null;
  }

  void send(Map<String, dynamic> msg) {
    if (_closedByUser) return;

    // si no está conectado, no explota: simplemente ignora (y la UI sigue)
    if (_channel == null) return;

    try {
      _channel!.sink.add(jsonEncode(msg));
      // si logramos mandar algo, asumimos que está vivo
      if (_state != DeskConnState.connected) {
        _setState(DeskConnState.connected);
      }
    } catch (_) {
      _scheduleReconnect();
    }
  }

  Future<Map<String, dynamic>> request(
    String type, {
    Map<String, dynamic>? payload,
    Duration timeout = const Duration(seconds: 10),
  }) {
    final id = '${DateTime.now().millisecondsSinceEpoch}_${_seq++}';
    final c = Completer<Map<String, dynamic>>();
    _pending[id] = c;

    send({
      'id': id,
      'type': type,
      if (payload != null) ...payload,
    });

    return c.future.timeout(timeout, onTimeout: () {
      _pending.remove(id);
      throw TimeoutException('Timeout esperando respuesta de $type');
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
