import 'package:flutter/material.dart';

class ConfigTab extends StatelessWidget {
  final double sensitivity;
  final double scrollSpeed;
  final int holdDelayMs;

  final ValueChanged<double> onSensitivityChanged;
  final ValueChanged<double> onScrollChanged;
  final ValueChanged<int> onHoldDelayChanged;

  final ThemeMode themeMode;
  final ValueChanged<ThemeMode> onThemeModeChanged;

  const ConfigTab({
    super.key,
    required this.sensitivity,
    required this.scrollSpeed,
    required this.holdDelayMs,
    required this.onSensitivityChanged,
    required this.onScrollChanged,
    required this.onHoldDelayChanged,
    required this.themeMode,
    required this.onThemeModeChanged,
  });

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text("Mouse", style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
        const SizedBox(height: 10),

        Text("Sensibilidad: ${sensitivity.toStringAsFixed(2)}"),
        Slider(
          value: sensitivity.clamp(0.2, 3.0),
          min: 0.2,
          max: 3.0,
          onChanged: onSensitivityChanged,
        ),

        const SizedBox(height: 10),
        Text("Scroll speed: ${scrollSpeed.toStringAsFixed(2)}"),
        Slider(
          value: scrollSpeed.clamp(0.2, 3.0),
          min: 0.2,
          max: 3.0,
          onChanged: onScrollChanged,
        ),

        const SizedBox(height: 10),
        Text("Hold delay (ms): $holdDelayMs"),
        Slider(
          value: holdDelayMs.toDouble().clamp(150.0, 900.0),
          min: 150,
          max: 900,
          divisions: 75, // pasos de ~10ms-12ms
          label: "$holdDelayMs ms",
          onChanged: (v) => onHoldDelayChanged(v.round()),
        ),

        const Divider(height: 32),

        const Text("Tema", style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
        const SizedBox(height: 10),

        RadioListTile<ThemeMode>(
          title: const Text("System"),
          value: ThemeMode.system,
          groupValue: themeMode,
          onChanged: (v) => v != null ? onThemeModeChanged(v) : null,
        ),
        RadioListTile<ThemeMode>(
          title: const Text("Light"),
          value: ThemeMode.light,
          groupValue: themeMode,
          onChanged: (v) => v != null ? onThemeModeChanged(v) : null,
        ),
        RadioListTile<ThemeMode>(
          title: const Text("Dark"),
          value: ThemeMode.dark,
          groupValue: themeMode,
          onChanged: (v) => v != null ? onThemeModeChanged(v) : null,
        ),
      ],
    );
  }
}
