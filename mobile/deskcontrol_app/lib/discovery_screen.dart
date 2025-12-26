import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'control_home.dart';
import 'desk_socket.dart';

class FoundHost {
  final InternetAddress address;
  final int wsPort;
  final String name;

  FoundHost(this.address, this.wsPort, this.name);
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

  @override
  void initState() {
    super.initState();
    _startDiscovery();
  }

  @override
  void dispose() {
    _stopDiscovery();
    super.dispose();
  }

  Future<void> _startDiscovery() async {
    _udp?.close();
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

    _broadcastTimer?.cancel();
    _broadcastTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      _sendDiscover();
    });

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
    final bytes = utf8.encode(msg);
    _udp!.send(bytes, InternetAddress('255.255.255.255'), discoveryPort);
  }

  Future<void> _connectAndOpen(FoundHost host) async {
    final url = 'ws://${host.address.address}:${host.wsPort}/ws';

    try {
      final desk = DeskSocket.connect(url);

      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => ControlHome(
            hostName: host.name,
            hostAddr: '${host.address.address}:${host.wsPort}',
            desk: desk,
            themeMode: widget.themeMode,
            onThemeModeChanged: widget.onThemeModeChanged,
          ),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('No pude conectar: $e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final entries = _found.entries.toList()
      ..sort((a, b) => a.value.name.compareTo(b.value.name));

    return Scaffold(
      appBar: AppBar(
        title: const Text('DeskControl - Dispositivos'),
        actions: [
          IconButton(
            tooltip: 'Refrescar',
            onPressed: () {
              setState(() => _found.clear());
              _sendDiscover();
            },
            icon: const Icon(Icons.refresh),
          )
        ],
      ),
      body: entries.isEmpty
          ? const Center(
              child: Text(
                'Buscando PCs...\n(asegúrate de estar en la misma Wi-Fi)\n\nSi no aparece: revisa firewall UDP 54546 y TCP 54545.',
                textAlign: TextAlign.center,
              ),
            )
          : ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: entries.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (context, i) {
                final h = entries[i].value;
                return Card(
                  child: ListTile(
                    title: Text(h.name),
                    subtitle: Text('${h.address.address}:${h.wsPort}'),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => _connectAndOpen(h),
                  ),
                );
              },
            ),
    );
  }
}
