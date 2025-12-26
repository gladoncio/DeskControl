import 'package:flutter/material.dart';
import '../desk_socket.dart';

class SoundTab extends StatelessWidget {
  final DeskSocket desk;
  const SoundTab({super.key, required this.desk});

  void _k(String key) => desk.send({"type": "key", "key": key});

  @override
  Widget build(BuildContext context) {
    Widget btn(IconData icon, String label, VoidCallback onTap) {
      return Expanded(
        child: SizedBox(
          height: 56,
          child: ElevatedButton.icon(
            onPressed: onTap,
            icon: Icon(icon),
            label: Text(label),
          ),
        ),
      );
    }

    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          const Text("Sonido (rápido)", style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          const Text("Por ahora: volumen master + multimedia. (Mezclador/mic cuando agreguemos WASAPI)."),
          const SizedBox(height: 12),
          Row(
            children: [
              btn(Icons.volume_off, "Mute", () => _k("vol_mute")),
              const SizedBox(width: 8),
              btn(Icons.volume_down, "Vol -", () => _k("vol_down")),
              const SizedBox(width: 8),
              btn(Icons.volume_up, "Vol +", () => _k("vol_up")),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              btn(Icons.skip_previous, "Prev", () => _k("media_prev")),
              const SizedBox(width: 8),
              btn(Icons.play_arrow, "Play/Pause", () => _k("media_play_pause")),
              const SizedBox(width: 8),
              btn(Icons.skip_next, "Next", () => _k("media_next")),
            ],
          ),
        ],
      ),
    );
  }
}
