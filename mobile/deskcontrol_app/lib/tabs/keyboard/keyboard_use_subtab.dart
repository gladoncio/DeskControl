import 'package:flutter/material.dart';
import '../../storage.dart';

class KeyboardUseSubTab extends StatelessWidget {
  final List<KeyItem> keys;
  final List<ComboItem> combos;

  final TextEditingController textCtrl;
  final VoidCallback onSendText;

  final void Function(KeyItem k) onKeyTap;
  final void Function(ComboItem c) onRunCombo;

  const KeyboardUseSubTab({
    super.key,
    required this.keys,
    required this.combos,
    required this.textCtrl,
    required this.onSendText,
    required this.onKeyTap,
    required this.onRunCombo,
  });

  @override
  Widget build(BuildContext context) {
    final map = {for (final k in keys) k.id: k};

    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        const Text(
          "Panel",
          style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 10),

        TextField(
          controller: textCtrl,
          minLines: 1,
          maxLines: 4,
          decoration: InputDecoration(
            labelText: "Escribe y envía",
            border: const OutlineInputBorder(),
            suffixIcon: IconButton(
              icon: const Icon(Icons.send),
              onPressed: onSendText,
            ),
          ),
          onSubmitted: (_) => onSendText(),
        ),

        const SizedBox(height: 16),
        const Divider(),
        const Text("Teclas", style: TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),

        if (keys.isEmpty)
          const Text("Aún no agregas teclas. Ve a Administrar para crear las tuyas.")
        else
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: keys.map((k) {
              return ElevatedButton(
                onPressed: () => onKeyTap(k),
                child: Text(k.name, textAlign: TextAlign.center),
              );
            }).toList(),
          ),

        const SizedBox(height: 16),
        const Divider(),
        const Text("Combinaciones", style: TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(height: 8),

        if (combos.isEmpty)
          const Text("Aún no agregas combos.")
        else
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: combos.map((c) {
              final names = c.keys.map((id) => map[id]?.name ?? "?").join(" + ");
              return ElevatedButton(
                onPressed: () => onRunCombo(c),
                child: Text("${c.name}\n$names", textAlign: TextAlign.center),
              );
            }).toList(),
          ),
      ],
    );
  }
}
