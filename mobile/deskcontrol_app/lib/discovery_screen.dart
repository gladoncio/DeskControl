import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';

import 'connection_config.dart';
import 'connection_settings_screen.dart';
import 'control_home.dart';
import 'desk_socket.dart';
import 'qr_scan_screen.dart';
import 'storage.dart';

class FoundHost {
  final InternetAddress address;
  final int wsPort;
  final String name;

  FoundHost(this.address, this.wsPort, this.name);
}

class _PairData {
  final String host;
  final int port;
  final bool tls;
  final String token;
  final String fp;

  const _PairData({
    required this.host,
    required this.port,
    required this.tls,
    required this.token,
    required this.fp,
  });
}

class DiscoveryScreen extends StatefulWidget {
  final ThemeMode themeMode;
  final ValueChanged<ThemeMode> onThemeModeChanged;

  const DiscoveryScreen({
    super.key,
    required this.themeMode,
    required this.onThemeModeChanged,
  });

  @override
  State<DiscoveryScreen> createState() => _DiscoveryScreenState();
}

class _DiscoveryScreenState extends State<DiscoveryScreen> {
  RawDatagramSocket? _udp;
  Timer? _broadcastTimer;
  final Map<String, FoundHost> _found = {};

  static const int wsDefaultPort = 54545;
  static const int discoveryPort = 54546;

  ConnectionData? _conn;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _loadConn();
  }

  @override
  void dispose() {
    _stopDiscovery();
    super.dispose();
  }

  Future<void> _loadConn() async {
    final c = await AppStorage.loadConnection();
    _conn = c;
    if (!mounted) return;
    setState(() => _loading = false);

    if (c.autoDiscover) {
      await _startDiscovery();
    } else {
      _stopDiscovery();
    }
  }

  Future<void> _openSettings() async {
    final res = await Navigator.of(context).push<ConnectionData>(
      MaterialPageRoute(builder: (_) => const ConnectionSettingsScreen()),
    );

    if (res == null) return;

    setState(() {
      _conn = res;
      _found.clear();
    });

    if (res.autoDiscover) {
      await _startDiscovery();
    } else {
      _stopDiscovery();
    }
  }

  _PairData? _parsePair(String raw) {
    // Esperado:
    // deskcontrol://pair?host=...&port=54545&tls=1&token=...&fp=HEX
    try {
      final u = Uri.parse(raw);
      if (u.scheme == 'deskcontrol' && u.host == 'pair') {
        final host = (u.queryParameters['host'] ?? '').trim();
        final port = int.tryParse((u.queryParameters['port'] ?? '').trim()) ?? 54545;
        final tls = (u.queryParameters['tls'] ?? '0').trim() == '1';
        final token = (u.queryParameters['token'] ?? '').trim();
        final fp = (u.queryParameters['fp'] ?? '').trim();

        if (host.isEmpty) return null;
        if (tls && (token.isEmpty || fp.isEmpty)) return null;

        return _PairData(host: host, port: port, tls: tls, token: token, fp: fp);
      }
    } catch (_) {}
    return null;
  }

  Future<void> _scanQrAndApply() async {
    final raw = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (_) => const QrScanScreen()),
    );
    if (raw == null || raw.trim().isEmpty) return;

    final pair = _parsePair(raw.trim());
    if (pair == null) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text("QR inválido (deskcontrol://pair...)")),
      );
      return;
    }

    final current = _conn ?? await AppStorage.loadConnection();

    final updated = current.copyWith(
      autoDiscover: false,
      host: pair.host,
      port: pair.port,
      useTls: pair.tls,
      token: pair.token,
      certFpSha256Hex: pair.fp,
      // cuenta se configura en settings, no en QR
    );

    await AppStorage.saveConnection(updated);
    if (!mounted) return;

    setState(() {
      _conn = updated;
      _found.clear();
    });

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text("QR OK: ${pair.host}:${pair.port}  TLS:${pair.tls ? 'ON' : 'OFF'}")),
    );
  }

  Future<void> _startDiscovery() async {
    _stopDiscovery();

    _udp = await RawDatagramSocket.bind(InternetAddress.anyIPv4, 0);
    _udp!.broadcastEnabled = true;

    _udp!.listen((event) {
      if (event != RawSocketEvent.read) return;
      final dg = _udp!.receive();
      if (dg == null) return;

      try {
        final data = utf8.decode(dg.data);
        final obj = jsonDecode(data);
        if (obj is! Map) return;
        if (obj['type'] != 'announce') return;
        if (obj['app'] != 'deskcontrol') return;

        final name = (obj['name'] ?? 'DeskControl-PC').toString();
        final wsPort =
            (obj['ws_port'] is int) ? obj['ws_port'] as int : wsDefaultPort;

        final ip = dg.address;
        final key = '${ip.address}:$wsPort';
        _found[key] = FoundHost(ip, wsPort, name);

        setState(() {});
      } catch (_) {}
    });

    _broadcastTimer =
        Timer.periodic(const Duration(seconds: 1), (_) => _sendDiscover());
    _sendDiscover();
  }

  void _stopDiscovery() {
    _broadcastTimer?.cancel();
    _broadcastTimer = null;
    _udp?.close();
    _udp = null;
  }

  void _sendDiscover() {
    if (_udp == null) return;
    final msg = jsonEncode({"type": "discover", "app": "deskcontrol", "v": 1});
    _udp!.send(
      utf8.encode(msg),
      InternetAddress('255.255.255.255'),
      discoveryPort,
    );
  }

  String _buildWsUrl(ConnectionData c, String host, int port) {
    final scheme = c.useTls ? 'wss' : 'ws';
    final qp = <String, String>{};
    // token también en query (además de header) para compatibilidad
    final t = c.token.trim();
    if (t.isNotEmpty) qp['token'] = t;
    final q = qp.isEmpty ? '' : '?${Uri(queryParameters: qp).query}';
    return '$scheme://$host:$port/ws$q';
  }

  Future<void> _connectUrl(String hostName, String hostAddr, String url) async {
    final c = _conn ?? await AppStorage.loadConnection();

    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => const AlertDialog(
        content: Row(
          children: [
            CircularProgressIndicator(),
            SizedBox(width: 12),
            Expanded(child: Text("Conectando…")),
          ],
        ),
      ),
    );

    final desk = DeskSocket.connect(
      url,
      useTls: c.useTls,
      certFpSha256Hex: c.certFpSha256Hex,
      securePayload: c.securePayload,
      token: c.token,
      useUserPass: c.useUserPass,
      username: c.username,
      password: c.password,
    );

    try {
      await desk.ready;
      if (!mounted) return;
      Navigator.of(context).pop();

      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => ControlHome(
            hostName: hostName,
            hostAddr: hostAddr,
            desk: desk,
            themeMode: widget.themeMode,
            onThemeModeChanged: widget.onThemeModeChanged,
          ),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      Navigator.of(context).pop();
      desk.close();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("No se pudo conectar: $e")),
      );
    }
  }

  Future<void> _connectFound(FoundHost h) async {
    final c = _conn ?? await AppStorage.loadConnection();
    final url = _buildWsUrl(c, h.address.address, h.wsPort);
    await _connectUrl(h.name, '${h.address.address}:${h.wsPort}', url);
  }

  Future<void> _connectManual() async {
    final c = _conn ?? await AppStorage.loadConnection();
    final host = c.host.trim();
    final port = c.port;

    if (host.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text("Configura IP/Host en Settings o escanea QR")),
      );
      return;
    }

    final url = _buildWsUrl(c, host, port);
    await _connectUrl(host, '$host:$port', url);
  }

  @override
  Widget build(BuildContext context) {
    final items = _found.values.toList()
      ..sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));

    final c = _conn;

    return Scaffold(
      appBar: AppBar(
        title: const Text('DeskControl'),
        actions: [
          IconButton(
            tooltip: "Escanear QR",
            onPressed: _scanQrAndApply,
            icon: const Icon(Icons.qr_code_scanner),
          ),
          IconButton(
            tooltip: "Settings de conexión",
            onPressed: _openSettings,
            icon: const Icon(Icons.settings),
          ),
          PopupMenuButton<String>(
            onSelected: (v) {
              if (v == 'system') widget.onThemeModeChanged(ThemeMode.system);
              if (v == 'light') widget.onThemeModeChanged(ThemeMode.light);
              if (v == 'dark') widget.onThemeModeChanged(ThemeMode.dark);
            },
            itemBuilder: (_) => const [
              PopupMenuItem(value: 'system', child: Text('Tema: Sistema')),
              PopupMenuItem(value: 'light', child: Text('Tema: Claro')),
              PopupMenuItem(value: 'dark', child: Text('Tema: Oscuro')),
            ],
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(12),
              children: [
                Card(
                  child: ListTile(
                    title: const Text("Conectar manual"),
                    subtitle: Text(
                      (c == null || c.host.trim().isEmpty)
                          ? "Configura IP/Host en Settings o escanea QR"
                          : "${c.host}:${c.port}  •  TLS:${c.useTls ? 'ON' : 'OFF'}  •  token:${c.token.trim().isEmpty ? '-' : 'OK'}  •  fp:${c.certFpSha256Hex.trim().isEmpty ? '-' : 'OK'}",
                    ),
                    trailing: ElevatedButton(
                      onPressed: _connectManual,
                      child: const Text("Conectar"),
                    ),
                  ),
                ),
                const SizedBox(height: 12),
                if (c != null && c.autoDiscover)
                  const Padding(
                    padding: EdgeInsets.only(bottom: 8),
                    child: Text("PCs encontrados (UDP):"),
                  ),
                ...items.map(
                  (h) => Card(
                    child: ListTile(
                      title: Text(h.name),
                      subtitle: Text("${h.address.address}:${h.wsPort}"),
                      trailing: ElevatedButton(
                        onPressed: () => _connectFound(h),
                        child: const Text("Conectar"),
                      ),
                    ),
                  ),
                ),
                if (c != null && c.autoDiscover && items.isEmpty)
                  const Padding(
                    padding: EdgeInsets.only(top: 16),
                    child: Center(child: Text("Buscando en la red…")),
                  ),
              ],
            ),
    );
  }
}
