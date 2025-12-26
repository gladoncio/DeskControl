import 'package:flutter/material.dart';
import 'discovery_screen.dart';

void main() => runApp(const DeskControlApp());

class DeskControlApp extends StatefulWidget {
  const DeskControlApp({super.key});

  @override
  State<DeskControlApp> createState() => _DeskControlAppState();
}

class _DeskControlAppState extends State<DeskControlApp> {
  ThemeMode _themeMode = ThemeMode.system; // system | light | dark

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'DeskControl',
      themeMode: _themeMode,
      theme: ThemeData(
        useMaterial3: true,
        brightness: Brightness.light,
      ),
      darkTheme: ThemeData(
        useMaterial3: true,
        brightness: Brightness.dark,
      ),
      home: DiscoveryScreen(
        themeMode: _themeMode,
        onThemeModeChanged: (m) => setState(() => _themeMode = m),
      ),
    );
  }
}
