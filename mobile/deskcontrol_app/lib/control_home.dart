import 'dart:async';
import 'package:flutter/material.dart';
import 'desk_socket.dart';
import 'storage.dart';

import 'tabs/mouse_tab.dart';
import 'tabs/keyboard_tab.dart';
import 'tabs/config_tab.dart';
import 'tabs/apps_tab.dart';
import 'tabs/sound_tab.dart';

class ControlHome extends StatefulWidget {
  final String hostName;
  final String hostAddr;
  final DeskSocket desk;

  final ThemeMode themeMode;
  final ValueChanged<ThemeMode> onThemeModeChanged;

  const ControlHome({
    super.key,
    required this.hostName,
    required this.hostAddr,
    required this.desk,
    required this.themeMode,
    required this.onThemeModeChanged,
  });

  @override
  State<ControlHome> createState() => _ControlHomeState();
}

class _ControlHomeState extends State<ControlHome> {
  late StreamSubscription _connSub;

  DeskConnState _state = DeskConnState.connecting;

  double sensitivity = 1.0;
  double scrollSpeed = 1.0;
  int holdDelayMs = 350;

  bool _shownDisconnectToast = false;

  @override
  void initState() {
    super.initState();
    _loadConfig();

    _state = widget.desk.state;
    _connSub = widget.desk.connection.listen((s) {
      if (!mounted) return;
      setState(() => _state = s);

      // Aviso claro si se cae (una vez)
      if ((s == DeskConnState.disconnected || s == DeskConnState.closed) &&
          !_shownDisconnectToast) {
        _shownDisconnectToast = true;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Text("Conexión perdida"),
            action: SnackBarAction(
              label: "Volver",
              onPressed: _disconnectAndBack,
            ),
            duration: const Duration(seconds: 6),
          ),
        );
      }

      if (s == DeskConnState.connected) {
        _shownDisconnectToast = false;
      }
    });
  }

  Future<void> _loadConfig() async {
    final cfg = await AppStorage.loadConfig();
    if (!mounted) return;
    setState(() {
      sensitivity = cfg.sensitivity;
      scrollSpeed = cfg.scrollSpeed;
      holdDelayMs = cfg.holdDelayMs;
    });
  }

  Future<void> _saveConfig() async {
    final tm = widget.themeMode == ThemeMode.dark
        ? "dark"
        : widget.themeMode == ThemeMode.light
            ? "light"
            : "system";

    await AppStorage.saveConfig(
      sensitivity: sensitivity,
      scrollSpeed: scrollSpeed,
      themeMode: tm,
      holdDelayMs: holdDelayMs,
    );
  }

  @override
  void dispose() {
    _connSub.cancel();
    widget.desk.close();
    super.dispose();
  }

  void _disconnectAndBack() {
    widget.desk.close();
    Navigator.of(context).pop();
  }

  String _statusText() {
    switch (_state) {
      case DeskConnState.connected:
        return "Conectado";
      case DeskConnState.connecting:
        return "Conectando...";
      case DeskConnState.reconnecting:
        return "Reconectando...";
      case DeskConnState.disconnected:
        return "Desconectado";
      case DeskConnState.closed:
        return "Cerrado";
    }
  }

  Color? _statusColor(ThemeData theme) {
    switch (_state) {
      case DeskConnState.connected:
        return Colors.green;
      case DeskConnState.reconnecting:
      case DeskConnState.connecting:
        return Colors.orange;
      case DeskConnState.disconnected:
      case DeskConnState.closed:
        return theme.colorScheme.error;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return DefaultTabController(
      length: 5,
      child: Scaffold(
        appBar: AppBar(
          title: Text(widget.hostName),
          bottom: const TabBar(
            isScrollable: true,
            tabs: [
              Tab(icon: Icon(Icons.mouse), text: 'Mouse'),
              Tab(icon: Icon(Icons.keyboard), text: 'Teclado'),
              Tab(icon: Icon(Icons.window), text: 'Apps'),
              Tab(icon: Icon(Icons.volume_up), text: 'Sonido'),
              Tab(icon: Icon(Icons.settings), text: 'Config'),
            ],
          ),
          actions: [
            Center(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Row(
                  children: [
                    Icon(Icons.circle, size: 10, color: _statusColor(theme)),
                    const SizedBox(width: 6),
                    Text(_statusText(), style: const TextStyle(fontSize: 12)),
                  ],
                ),
              ),
            ),
            IconButton(
              tooltip: 'Desconectar',
              onPressed: _disconnectAndBack,
              icon: const Icon(Icons.link_off),
            )
          ],
        ),
        body: TabBarView(
          physics: const NeverScrollableScrollPhysics(),
          children: [
            MouseTab(
              desk: widget.desk,
              sensitivity: sensitivity,
              scrollSpeed: scrollSpeed,
              holdDelayMs: holdDelayMs,
            ),
            KeyboardTab(desk: widget.desk),
            AppsTab(desk: widget.desk),
            SoundTab(desk: widget.desk),
            ConfigTab(
              sensitivity: sensitivity,
              scrollSpeed: scrollSpeed,
              holdDelayMs: holdDelayMs,
              onSensitivityChanged: (v) async {
                setState(() => sensitivity = v);
                await _saveConfig();
              },
              onScrollChanged: (v) async {
                setState(() => scrollSpeed = v);
                await _saveConfig();
              },
              onHoldDelayChanged: (ms) async {
                setState(() => holdDelayMs = ms);
                await _saveConfig();
              },
              themeMode: widget.themeMode,
              onThemeModeChanged: (m) async {
                widget.onThemeModeChanged(m);
                await _saveConfig();
              },
            ),
          ],
        ),
      ),
    );
  }
}
