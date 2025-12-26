import 'dart:async';
import 'package:flutter/material.dart';
import '../../storage.dart';

class KeyboardAdminSubTab extends StatelessWidget {
  final List<KeyItem> keys;
  final List<ComboItem> combos;

  final Set<String> visibleKeyIds;

  // ✅ Si una tecla está en un combo, aquí entra y queda “siempre visible”
  final Set<String> forcedVisibleKeyIds;

  final Future<void> Function(KeyItem k, bool v) onToggleVisible;

  final Future<void> Function() onAddTextKey;
  final Future<void> Function() onCaptureVK;

  final Future<void> Function(KeyItem k) onDeleteKey;

  final Future<void> Function() onCreateCombo;
  final Future<void> Function(ComboItem c) onDeleteCombo;
  final Future<void> Function(ComboItem c) onEditComboTapIndex;

  const KeyboardAdminSubTab({
    super.key,
    required this.keys,
    required this.combos,
    required this.visibleKeyIds,
    required this.forcedVisibleKeyIds,
    required this.onToggleVisible,
    required this.onAddTextKey,
    required this.onCaptureVK,
    required this.onDeleteKey,
    required this.onCreateCombo,
    required this.onDeleteCombo,
    required this.onEditComboTapIndex,
  });

  @override
  Widget build(BuildContext context) {
    final map = {for (final k in keys) k.id: k};
    bool isForced(String id) => forcedVisibleKeyIds.contains(id);

    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                "Administrar",
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
              ),
            ),
            IconButton(
              tooltip: "Agregar tecla (texto)",
              onPressed: () async => onAddTextKey(),
              icon: const Icon(Icons.text_fields),
            ),
            IconButton(
              tooltip: "Capturar tecla (VK)",
              onPressed: () async => onCaptureVK(),
              icon: const Icon(Icons.radio_button_checked),
            ),
          ],
        ),
        const SizedBox(height: 12),
        const Text(
          "Teclas guardadas (elige cuáles se ven en User):",
          style: TextStyle(fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 8),

        if (keys.isEmpty)
          const Text("No tienes teclas todavía.")
        else
          ...keys.map((k) {
            final forced = isForced(k.id);
            final isVisible = forced || visibleKeyIds.contains(k.id);

            final detail = k.useVK
                ? "VK:${k.keySpec?.vk} SC:${k.keySpec?.scan}${(k.keySpec?.ext ?? false) ? ' EXT' : ''}"
                : (k.keyName ?? "");

            final subtitle = forced
                ? "$detail\n(Usada en combo: siempre visible)"
                : detail;

            return Card(
              child: ListTile(
                title: Text(k.name),
                subtitle: Text(subtitle),
                leading: Switch(
                  value: isVisible,
                  // ✅ Si está usada en combo: no se puede ocultar
                  onChanged: forced ? null : (v) async => onToggleVisible(k, v),
                ),
                trailing: PopupMenuButton<String>(
                  onSelected: (v) async {
                    if (v == 'del') await onDeleteKey(k);
                  },
                  itemBuilder: (_) => const [
                    PopupMenuItem(value: 'del', child: Text("Eliminar")),
                  ],
                ),
              ),
            );
          }),

        const SizedBox(height: 16),
        const Divider(),

        Row(
          children: [
            const Expanded(
              child: Text(
                "Combos",
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
              ),
            ),
            IconButton(
              tooltip: "Crear combo",
              onPressed: () async => onCreateCombo(),
              icon: const Icon(Icons.add),
            ),
          ],
        ),
        const SizedBox(height: 8),

        if (combos.isEmpty)
          const Text("No tienes combos todavía.")
        else
          ...combos.map((c) {
            final names = c.keys.map((id) => map[id]?.name ?? "?").join(" + ");
            final tapName = c.keys.isEmpty
                ? "—"
                : (map[c.keys[c.tapIndex.clamp(0, c.keys.length - 1)]]?.name ?? "?");

            return Card(
              child: ListTile(
                title: Text(c.name),
                subtitle: Text("$names\nTap: $tapName"),
                isThreeLine: true,
                trailing: PopupMenuButton<String>(
                  onSelected: (v) async {
                    if (v == 'tap') await onEditComboTapIndex(c);
                    if (v == 'del') await onDeleteCombo(c);
                  },
                  itemBuilder: (_) => const [
                    PopupMenuItem(value: 'tap', child: Text("Elegir Tap")),
                    PopupMenuItem(value: 'del', child: Text("Eliminar")),
                  ],
                ),
              ),
            );
          }),
      ],
    );
  }
}

// (si ya lo tenías, déjalo igual; si no, también compila así)
class CaptureCountdownDialog extends StatefulWidget {
  final int seconds;
  final Future<dynamic> done;
  final VoidCallback? onCancel;

  const CaptureCountdownDialog({
    super.key,
    required this.seconds,
    required this.done,
    this.onCancel,
  });

  @override
  State<CaptureCountdownDialog> createState() => _CaptureCountdownDialogState();
}

class _CaptureCountdownDialogState extends State<CaptureCountdownDialog> {
  late int _left;
  Timer? _t;
  bool _closed = false;

  @override
  void initState() {
    super.initState();
    _left = widget.seconds;

    widget.done.whenComplete(() {
      if (mounted && !_closed) {
        _closed = true;
        Navigator.of(context).pop(false);
      }
    });

    _t = Timer.periodic(const Duration(seconds: 1), (t) {
      if (!mounted) return;
      setState(() => _left = (_left - 1).clamp(0, widget.seconds));
      if (_left <= 0) t.cancel();
    });
  }

  @override
  void dispose() {
    _t?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text("Presiona una tecla"),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            _left.toString(),
            style: const TextStyle(fontSize: 64, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 8),
          Text(
            _left > 0
                ? "Tienes $_left s para presionar una tecla en el PC"
                : "Esperando respuesta…",
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 12),
          const LinearProgressIndicator(),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () {
            if (_closed) return;
            _closed = true;
            widget.onCancel?.call();
            Navigator.of(context).pop(true);
          },
          child: const Text("Cancelar"),
        ),
      ],
    );
  }
}
