import 'package:flutter/material.dart';
import '../desk_socket.dart';

class ConfigTab extends StatefulWidget {
  final DeskSocket desk;
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
    required this.desk,
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
  State<ConfigTab> createState() => _ConfigTabState();
}

class _ConfigTabState extends State<ConfigTab> {
  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text("Mouse",
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
        const SizedBox(height: 10),

        Text("Sensibilidad: ${widget.sensitivity.toStringAsFixed(2)}"),
        Slider(
          value: widget.sensitivity.clamp(0.2, 3.0),
          min: 0.2,
          max: 3.0,
          onChanged: widget.onSensitivityChanged,
        ),

        const SizedBox(height: 10),
        Text("Scroll speed: ${widget.scrollSpeed.toStringAsFixed(2)}"),
        Slider(
          value: widget.scrollSpeed.clamp(0.2, 3.0),
          min: 0.2,
          max: 3.0,
          onChanged: widget.onScrollChanged,
        ),

        const SizedBox(height: 10),
        Text("Hold delay (ms): ${widget.holdDelayMs}"),
        Slider(
          value: widget.holdDelayMs.toDouble().clamp(150.0, 900.0),
          min: 150,
          max: 900,
          divisions: 75,
          label: "${widget.holdDelayMs} ms",
          onChanged: (v) => widget.onHoldDelayChanged(v.round()),
        ),

        const Divider(height: 32),

        const Text("Tema",
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
        const SizedBox(height: 10),

        RadioListTile<ThemeMode>(
          title: const Text("System"),
          value: ThemeMode.system,
          groupValue: widget.themeMode,
          onChanged: (v) => v != null ? widget.onThemeModeChanged(v) : null,
        ),
        RadioListTile<ThemeMode>(
          title: const Text("Light"),
          value: ThemeMode.light,
          groupValue: widget.themeMode,
          onChanged: (v) => v != null ? widget.onThemeModeChanged(v) : null,
        ),
        RadioListTile<ThemeMode>(
          title: const Text("Dark"),
          value: ThemeMode.dark,
          groupValue: widget.themeMode,
          onChanged: (v) => v != null ? widget.onThemeModeChanged(v) : null,
        ),

        const SizedBox(height: 24),
        Center(
          child: Text(
            "DeskControl v1.2.0",
            style: Theme.of(context)
                .textTheme
                .bodySmall
                ?.copyWith(color: Colors.grey),
          ),
        ),
        const SizedBox(height: 8),
      ],
    );
  }
}
