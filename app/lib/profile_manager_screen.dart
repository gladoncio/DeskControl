import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'storage.dart';

class ProfileManagerScreen extends StatefulWidget {
  const ProfileManagerScreen({super.key});

  @override
  State<ProfileManagerScreen> createState() => _ProfileManagerScreenState();
}

class _ProfileManagerScreenState extends State<ProfileManagerScreen> {
  List<ConnectionRecord> _records = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final r = await AppStorage.loadConnectionHistory();
    if (!mounted) return;
    setState(() {
      _records = r;
      _loading = false;
    });
  }

  Future<void> _rename(ConnectionRecord rec) async {
    final loaded = await AppStorage.loadProfileName(rec.profileId, deviceId: rec.deviceId);
    final currentName = loaded.isNotEmpty ? loaded : rec.name;
    final ctrl = TextEditingController(text: currentName);
    final newName = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("Renombrar perfil"),
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
            child: const Text("Guardar"),
          ),
        ],
      ),
    );
    if (newName == null) return;
    // Save under deviceId if available, so all IPs share the same alias
    await AppStorage.saveProfileName(rec.profileId, newName, deviceId: rec.deviceId);
  }

  Future<void> _delete(ConnectionRecord rec) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("Eliminar perfil"),
        content: Text(
            "¿Eliminar '${rec.name}' y todas sus teclas/combos?\n\n"
            "(${rec.profileId})"),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text("Cancelar"),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
                backgroundColor: Theme.of(ctx).colorScheme.error),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text("Eliminar"),
          ),
        ],
      ),
    );
    if (confirm != true) return;

    final sp = await SharedPreferences.getInstance();
    final prefix = 'profile:${rec.profileId}:';
    final allKeys = sp.getKeys();
    for (final k in allKeys) {
      if (k.startsWith(prefix)) {
        await sp.remove(k);
      }
    }
    await AppStorage.removeConnectionRecord(rec.profileId);
    await _load();
  }

  Future<void> _edit(ConnectionRecord rec) async {
    final hostCtrl = TextEditingController(text: rec.host);
    final portCtrl = TextEditingController(text: rec.port.toString());
    bool useTls = rec.useTls;

    final result = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDlgState) => AlertDialog(
          title: const Text("Editar perfil"),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: hostCtrl,
                decoration: const InputDecoration(
                  labelText: "IP / Host",
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: portCtrl,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: "Puerto",
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 10),
              SwitchListTile(
                title: const Text("TLS"),
                value: useTls,
                onChanged: (v) => setDlgState(() => useTls = v),
                contentPadding: EdgeInsets.zero,
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(null),
              child: const Text("Cancelar"),
            ),
            FilledButton(
              onPressed: () {
                final host = hostCtrl.text.trim();
                final port = int.tryParse(portCtrl.text.trim()) ?? 54545;
                if (host.isEmpty) return;
                Navigator.of(ctx).pop({
                  'host': host,
                  'port': port,
                  'useTls': useTls,
                });
              },
              child: const Text("Guardar"),
            ),
          ],
        ),
      ),
    );
    if (result == null) return;

    final newHost = result['host'] as String;
    final newPort = result['port'] as int;
    final newTls = result['useTls'] as bool;
    final newProfileId = '$newHost:$newPort';

    // Si cambió el profileId, migrar las keys/combos al nuevo perfil
    if (newProfileId != rec.profileId) {
      final oldKeys = await AppStorage.loadKeys(profileId: rec.profileId);
      final oldCombos = await AppStorage.loadCombos(profileId: rec.profileId);
      final oldVisible = await AppStorage.loadVisibleKeyIds(profileId: rec.profileId);
      if (oldKeys.isNotEmpty) {
        await AppStorage.saveKeys(oldKeys, profileId: newProfileId);
      }
      if (oldCombos.isNotEmpty) {
        await AppStorage.saveCombos(oldCombos, profileId: newProfileId);
      }
      if (oldVisible.isNotEmpty) {
        await AppStorage.saveVisibleKeyIds(oldVisible, profileId: newProfileId);
      }
      // Limpiar perfil antiguo
      final sp = await SharedPreferences.getInstance();
      final prefix = 'profile:${rec.profileId}:';
      for (final k in sp.getKeys()) {
        if (k.startsWith(prefix)) {
          await sp.remove(k);
        }
      }
      await AppStorage.removeConnectionRecord(rec.profileId);
    }

    await AppStorage.addConnectionRecord(ConnectionRecord(
      profileId: newProfileId,
      host: newHost,
      port: newPort,
      name: rec.name,
      useTls: newTls,
      lastConnected: rec.lastConnected,
    ));
    await _load();
  }

  Future<void> _export(ConnectionRecord rec) async {
    final jsonStr = await AppStorage.exportProfile(rec.profileId);
    if (jsonStr == null) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text("No hay teclas/combos para exportar")),
      );
      return;
    }
    await Clipboard.setData(ClipboardData(text: jsonStr));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text("Perfil copiado al portapapeles (JSON)")),
    );
  }

  Future<void> _import() async {
    final ctrl = TextEditingController();
    final jsonStr = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("Importar perfil"),
        content: TextField(
          controller: ctrl,
          maxLines: 6,
          decoration: const InputDecoration(
            labelText: "Pega el JSON del perfil",
            border: OutlineInputBorder(),
          ),
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
            child: const Text("Importar"),
          ),
        ],
      ),
    );
    if (jsonStr == null) return;

    try {
      final obj = jsonDecode(jsonStr);
      if (obj is! Map) throw FormatException("No es un objeto JSON");
      final profileId =
          (obj['profileId'] ?? '').toString();
      if (profileId.isEmpty) throw FormatException("Falta profileId");

      await AppStorage.importProfile(profileId, jsonStr);

      final host = profileId.split(':').first;
      final portStr = profileId.split(':').last;
      final port = int.tryParse(portStr) ?? 54545;

      await AppStorage.addConnectionRecord(ConnectionRecord(
        profileId: profileId,
        host: host,
        port: port,
        name: host,
        useTls: false,
        lastConnected: 0,
      ));

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
            content: Text("Perfil '$profileId' importado (${(obj['keys'] as List?)?.length ?? 0} teclas)")),
      );
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("Error al importar: $e")),
      );
    }
  }

  Future<String> _resolveName(ConnectionRecord rec) async {
    final custom = await AppStorage.loadProfileName(rec.profileId, deviceId: rec.deviceId);
    return custom.isNotEmpty ? custom : rec.name;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text("Perfiles"),
        actions: [
          IconButton(
            tooltip: "Importar perfil",
            onPressed: _import,
            icon: const Icon(Icons.file_download),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _records.isEmpty
              ? const Center(
                  child: Text("No hay perfiles guardados.\n"
                      "Conéctate a un PC para crear uno."),
                )
              : ListView.builder(
                  padding: const EdgeInsets.all(12),
                  itemCount: _records.length,
                  itemBuilder: (ctx, i) {
                    final rec = _records[i];
                    return FutureBuilder<String>(
                      future: _resolveName(rec),
                      builder: (ctx, snap) {
                        final displayName = snap.data ?? rec.name;
                        return Card(
                          child: ListTile(
                            leading: const Icon(Icons.computer),
                            title: Text(displayName),
                            subtitle: Text(
                              "${rec.host}:${rec.port}  •  "
                              "${rec.useTls ? 'TLS' : 'no TLS'}\n"
                              "${rec.profileId}",
                            ),
                            isThreeLine: true,
                              trailing: PopupMenuButton<String>(
                              onSelected: (v) async {
                                if (v == 'connect') {
                                  Navigator.of(context).pop(rec);
                                  return;
                                }
                                if (v == 'edit') await _edit(rec);
                                if (v == 'rename') await _rename(rec);
                                if (v == 'export') await _export(rec);
                                if (v == 'del') await _delete(rec);
                                await _load();
                              },
                              itemBuilder: (_) => [
                                const PopupMenuItem(
                                  value: 'connect',
                                  child: Text("Conectar"),
                                ),
                                const PopupMenuItem(
                                  value: 'edit',
                                  child: Text("Editar conexión"),
                                ),
                                const PopupMenuItem(
                                  value: 'rename',
                                  child: Text("Renombrar"),
                                ),
                                const PopupMenuItem(
                                  value: 'export',
                                  child: Text("Exportar (JSON)"),
                                ),
                                const PopupMenuItem(
                                  value: 'del',
                                  child: Text("Eliminar"),
                                ),
                              ],
                            ),
                          ),
                        );
                      },
                    );
                  },
                ),
    );
  }
}
