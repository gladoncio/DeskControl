import 'dart:math';
import 'package:flutter/material.dart';

import '../../desk_socket.dart';
import '../../storage.dart';

import 'keyboard_use_subtab.dart';
import 'keyboard_admin_subtab.dart';

class KeyboardTab extends StatefulWidget {
  final DeskSocket desk;
  const KeyboardTab({super.key, required this.desk});

  @override
  State<KeyboardTab> createState() => _KeyboardTabState();
}

class _KeyboardTabState extends State<KeyboardTab> {
  bool _loading = true;

  List<KeyItem> keys = [];
  List<ComboItem> combos = [];

  // ✅ Decide qué aparece en User
  Set<String> visibleKeyIds = {};

  final _textCtrl = TextEditingController();
  final _rnd = Random();

  @override
  void initState() {
    super.initState();
    _loadAll();
  }

  @override
  void dispose() {
    _textCtrl.dispose();
    super.dispose();
  }

  String _newId() =>
      "${DateTime.now().microsecondsSinceEpoch}_${_rnd.nextInt(1 << 32)}";

  Future<void> _loadAll() async {
    setState(() => _loading = true);

    final k = await AppStorage.loadKeys();
    final c = await AppStorage.loadCombos();
    final vis = await AppStorage.loadVisibleKeyIds();

    // Si no hay visibilidad guardada: por defecto todas visibles
    final vset = <String>{...vis};
    if (vset.isEmpty && k.isNotEmpty) {
      vset.addAll(k.map((e) => e.id));
      await AppStorage.saveVisibleKeyIds(vset.toList());
    }

    if (!mounted) return;
    setState(() {
      keys = k;
      combos = c;
      visibleKeyIds = vset;
      _loading = false;
    });
  }

  Future<void> _saveKeys() async => AppStorage.saveKeys(keys);
  Future<void> _saveCombos() async => AppStorage.saveCombos(combos);
  Future<void> _saveVisible() async =>
      AppStorage.saveVisibleKeyIds(visibleKeyIds.toList());

  // ------------------------------------------------------------
  // ✅ Envío con tu API real: DeskSocket.send(Map)
  // ------------------------------------------------------------
  void _sendMsg(Map<String, dynamic> obj) {
    widget.desk.send(obj);
  }

  // ✅ Root fields (compat con tu daemon)
  void _sendTextPayload(String text) {
    _sendMsg({
      "type": "text_input",
      "text": text,
    });
  }

  void _sendKeyDownPayload(int vk, int scan, bool ext) {
    _sendMsg({
      "type": "input_key_down",
      "vk": vk,
      "scan": scan,
      "ext": ext,
    });
  }

  void _sendKeyUpPayload(int vk, int scan, bool ext) {
    _sendMsg({
      "type": "input_key_up",
      "vk": vk,
      "scan": scan,
      "ext": ext,
    });
  }

  void _sendKeyTapPayload(int vk, int scan, bool ext) {
    _sendMsg({
      "type": "input_key_tap",
      "vk": vk,
      "scan": scan,
      "ext": ext,
    });
  }

  // ✅ Hotkey nativo del daemon (mejor para combos)
  void _sendHotkeyVK(List<String> mods, KeySpec key) {
    _sendMsg({
      "type": "hotkey_vk",
      "mods": mods,
      "key": {"vk": key.vk, "scan": key.scan, "ext": key.ext},
    });
  }

  // ------------------------------------------------------------
  // Acciones de teclas
  // ------------------------------------------------------------
  void _keyTap(KeyItem k) {
    if (k.useVK && k.keySpec != null) {
      _sendKeyTapPayload(k.keySpec!.vk, k.keySpec!.scan, k.keySpec!.ext);
    } else {
      final t = k.keyName ?? "";
      if (t.isNotEmpty) _sendTextPayload(t);
    }
  }

  void _keyDown(KeyItem k) {
    if (!k.useVK || k.keySpec == null) return;
    _sendKeyDownPayload(k.keySpec!.vk, k.keySpec!.scan, k.keySpec!.ext);
  }

  void _keyUp(KeyItem k) {
    if (!k.useVK || k.keySpec == null) return;
    _sendKeyUpPayload(k.keySpec!.vk, k.keySpec!.scan, k.keySpec!.ext);
  }

  void _sendText() {
    final t = _textCtrl.text.trim();
    if (t.isEmpty) return;
    _sendTextPayload(t);
    _textCtrl.clear();
  }

  // ------------------------------------------------------------
  // ✅ Combos
  // - Si es mods + 1 tecla -> usa hotkey_vk (1 solo mensaje, 1 solo SendInput)
  // - Si no -> fallback al método down/tap/up actual
  // ------------------------------------------------------------
  String? _vkToModName(int vk) {
    // Ctrl
    if (vk == 0x11 || vk == 0xA2 || vk == 0xA3) return "ctrl";
    // Alt (MENU)
    if (vk == 0x12 || vk == 0xA4 || vk == 0xA5) return "alt";
    // Shift
    if (vk == 0x10 || vk == 0xA0 || vk == 0xA1) return "shift";
    // Win
    if (vk == 0x5B || vk == 0x5C) return "win";
    return null;
  }

  void _runCombo(ComboItem c) {
    final map = {for (final k in keys) k.id: k};
    final resolved = <KeyItem>[];
    for (final id in c.keys) {
      final item = map[id];
      if (item != null) resolved.add(item);
    }
    if (resolved.isEmpty) return;

    // Intentamos modo "hotkey_vk": solo teclas VK
    final vkKeys = resolved.where((k) => k.useVK && k.keySpec != null).toList();
    if (vkKeys.length == resolved.length && vkKeys.isNotEmpty) {
      final mods = <String>[];
      final nonMods = <KeyItem>[];

      for (final k in vkKeys) {
        final vk = k.keySpec!.vk;
        final mod = _vkToModName(vk);
        if (mod != null) {
          mods.add(mod);
        } else {
          nonMods.add(k);
        }
      }

      // Si hay EXACTAMENTE 1 tecla principal y al menos 1 mod => hotkey_vk
      if (mods.isNotEmpty && nonMods.length == 1) {
        _sendHotkeyVK(mods, nonMods[0].keySpec!);
        return;
      }
    }

    // Fallback: down de todas (except tap) + tap + up (reverso)
    final ti = (c.tapIndex).clamp(0, resolved.length - 1);

    if (resolved.length == 1) {
      _keyTap(resolved[0]);
      return;
    }

    for (int i = 0; i < resolved.length; i++) {
      if (i == ti) continue;
      _keyDown(resolved[i]);
    }

    _keyTap(resolved[ti]);

    for (int i = resolved.length - 1; i >= 0; i--) {
      if (i == ti) continue;
      _keyUp(resolved[i]);
    }
  }

  // ---------------- Admin helpers ----------------
  Future<String?> _askName(String title, {String? hint}) async {
    final ctrl = TextEditingController(text: hint ?? "");
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: TextField(
          controller: ctrl,
          decoration: const InputDecoration(
            labelText: "Nombre",
            border: OutlineInputBorder(),
          ),
          autofocus: true,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(null),
            child: const Text("Cancelar"),
          ),
          FilledButton(
            onPressed: () {
              final v = ctrl.text.trim();
              Navigator.of(ctx).pop(v.isEmpty ? null : v);
            },
            child: const Text("OK"),
          ),
        ],
      ),
    );
  }

  Future<void> _toggleVisible(KeyItem k, bool v) async {
    setState(() {
      if (v) {
        visibleKeyIds.add(k.id);
      } else {
        visibleKeyIds.remove(k.id);
      }
    });
    await _saveVisible();
  }

  Future<void> _addTextKey() async {
    final name = await _askName("Nombre de la tecla");
    if (name == null) return;

    final text = await _askName("Texto a enviar", hint: name);
    if (text == null) return;

    final item = KeyItem(
      id: _newId(),
      name: name,
      useVK: false,
      keySpec: null,
      keyName: text,
    );

    setState(() {
      keys.add(item);
      visibleKeyIds.add(item.id);
    });

    await _saveKeys();
    await _saveVisible();
  }

  // 👇 Aquí deja tu lógica real de captura VK (la que ya tenías antes)
  Future<void> _addKeyCaptureVK() async {
    // TODO: tu lógica existente
  }

  Future<void> _deleteKey(KeyItem k) async {
    setState(() {
      keys.removeWhere((x) => x.id == k.id);
      visibleKeyIds.remove(k.id);

      combos = combos
          .map((c) => ComboItem(
                id: c.id,
                name: c.name,
                keys: c.keys.where((id) => id != k.id).toList(),
                tapIndex: c.tapIndex.clamp(
                    0,
                    max(
                      0,
                      (c.keys.where((id) => id != k.id).length - 1),
                    )),
              ))
          .where((c) => c.keys.isNotEmpty)
          .toList();
    });

    await _saveKeys();
    await _saveVisible();
    await _saveCombos();
  }

  Future<void> _createCombo() async {
    final name = await _askName("Nombre del combo");
    if (name == null) return;

    final c = ComboItem(id: _newId(), name: name, keys: const [], tapIndex: 0);

    setState(() => combos.add(c));
    await _saveCombos();
  }

  Future<void> _deleteCombo(ComboItem c) async {
    setState(() => combos.removeWhere((x) => x.id == c.id));
    await _saveCombos();
  }

  Future<void> _editComboTapIndex(ComboItem c) async {
    if (c.keys.isEmpty) return;
    final idx = await showDialog<int>(
      context: context,
      builder: (ctx) => SimpleDialog(
        title: const Text("Elegir Tap"),
        children: [
          for (int i = 0; i < c.keys.length; i++)
            SimpleDialogOption(
              onPressed: () => Navigator.of(ctx).pop(i),
              child: Text("Tap index: $i"),
            ),
        ],
      ),
    );
    if (idx == null) return;

    setState(() {
      combos = combos
          .map((x) => x.id == c.id
              ? ComboItem(id: x.id, name: x.name, keys: x.keys, tapIndex: idx)
              : x)
          .toList();
    });
    await _saveCombos();
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    // ✅ Si una tecla está en un combo, se fuerza a mostrarse en User
    final forcedVisibleKeyIds = <String>{
      for (final c in combos) ...c.keys,
    };

    final userVisibleIds = <String>{
      ...visibleKeyIds,
      ...forcedVisibleKeyIds,
    };

    final userKeys = keys.where((k) => userVisibleIds.contains(k.id)).toList();

    return DefaultTabController(
      length: 2,
      child: Column(
        children: [
          const Material(
            child: TabBar(
              tabs: [
                Tab(text: "Usar"),
                Tab(text: "Administrar"),
              ],
            ),
          ),
          Expanded(
            child: TabBarView(
              children: [
                KeyboardUseSubTab(
                  keys: userKeys,
                  combos: combos,
                  textCtrl: _textCtrl,
                  onSendText: _sendText,
                  onKeyTap: _keyTap,
                  onRunCombo: _runCombo,
                ),
                KeyboardAdminSubTab(
                  keys: keys,
                  combos: combos,
                  visibleKeyIds: visibleKeyIds,
                  forcedVisibleKeyIds: forcedVisibleKeyIds,
                  onToggleVisible: _toggleVisible,
                  onAddTextKey: _addTextKey,
                  onCaptureVK: _addKeyCaptureVK,
                  onDeleteKey: _deleteKey,
                  onCreateCombo: _createCombo,
                  onDeleteCombo: _deleteCombo,
                  onEditComboTapIndex: _editComboTapIndex,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
