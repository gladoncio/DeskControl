import 'package:flutter/material.dart';

import 'connection_config.dart';
import 'storage.dart';

class ConnectionSettingsScreen extends StatefulWidget {
  const ConnectionSettingsScreen({super.key});

  @override
  State<ConnectionSettingsScreen> createState() =>
      _ConnectionSettingsScreenState();
}

class _ConnectionSettingsScreenState extends State<ConnectionSettingsScreen> {
  bool _loading = true;

  bool _autoDiscover = true;
  final _hostCtrl = TextEditingController();
  final _portCtrl = TextEditingController(text: "54545");

  bool _useTls = false;
  final _tokenCtrl = TextEditingController();
  final _fpCtrl = TextEditingController();

  bool _useUserPass = false;
  final _userCtrl = TextEditingController();
  final _passCtrl = TextEditingController();

  bool _securePayload = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final c = await AppStorage.loadConnection();
    if (!mounted) return;
    setState(() {
      _autoDiscover = c.autoDiscover;
      _hostCtrl.text = c.host;
      _portCtrl.text = c.port.toString();

      _useTls = c.useTls;
      _tokenCtrl.text = c.token;
      _fpCtrl.text = c.certFpSha256Hex;

      _useUserPass = c.useUserPass;
      _userCtrl.text = c.username;
      _passCtrl.text = c.password;

      _securePayload = c.securePayload;

      _loading = false;
    });
  }

  Future<void> _save() async {
    final host = _hostCtrl.text.trim();
    final port = int.tryParse(_portCtrl.text.trim()) ?? 54545;

    // Cuenta solo con TLS ON (tu política)
    final useUserPass = _useTls ? _useUserPass : false;

    final c = ConnectionData(
      autoDiscover: _autoDiscover,
      host: host,
      port: port.clamp(1, 65535),

      useTls: _useTls,
      token: _tokenCtrl.text.trim(),
      certFpSha256Hex: _fpCtrl.text.trim(),

      useUserPass: useUserPass,
      username: _userCtrl.text.trim(),
      password: _passCtrl.text,

      securePayload: _securePayload,
    );

    await AppStorage.saveConnection(c);

    if (!mounted) return;
    Navigator.of(context).pop(c);
  }

  @override
  void dispose() {
    _hostCtrl.dispose();
    _portCtrl.dispose();
    _tokenCtrl.dispose();
    _fpCtrl.dispose();
    _userCtrl.dispose();
    _passCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text("Conexión"),
        actions: [
          TextButton(onPressed: _save, child: const Text("Guardar")),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          Card(
            child: Column(
              children: [
                SwitchListTile(
                  title: const Text("Auto-discover (UDP)"),
                  subtitle:
                      const Text("Lista PCs automáticamente en la red local"),
                  value: _autoDiscover,
                  onChanged: (v) => setState(() => _autoDiscover = v),
                ),
                const Divider(height: 1),
                const ListTile(
                  title: Text("Conexión manual"),
                  subtitle: Text("IP/Host + Puerto"),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                  child: Column(
                    children: [
                      TextField(
                        controller: _hostCtrl,
                        decoration: const InputDecoration(
                          labelText: "IP / Host",
                          hintText: "Ej: 192.168.18.113",
                          border: OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 10),
                      TextField(
                        controller: _portCtrl,
                        keyboardType: TextInputType.number,
                        decoration: const InputDecoration(
                          labelText: "Puerto",
                          hintText: "54545",
                          border: OutlineInputBorder(),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 12),

          Card(
            child: Column(
              children: [
                SwitchListTile(
                  title: const Text("TLS (WSS)"),
                  subtitle: const Text(
                      "Modo seguro: requiere token + fingerprint (QR)"),
                  value: _useTls,
                  onChanged: (v) => setState(() {
                    _useTls = v;
                    if (!v) _useUserPass = false;
                  }),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                  child: Column(
                    children: [
                      TextField(
                        controller: _tokenCtrl,
                        enabled: _useTls,
                        decoration: const InputDecoration(
                          labelText: "Token (QR)",
                          border: OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 10),
                      TextField(
                        controller: _fpCtrl,
                        enabled: _useTls,
                        decoration: const InputDecoration(
                          labelText: "Fingerprint SHA-256 (hex) (QR)",
                          border: OutlineInputBorder(),
                        ),
                      ),
                      if (_useTls &&
                          (_tokenCtrl.text.trim().isEmpty ||
                              _fpCtrl.text.trim().isEmpty))
                        const Padding(
                          padding: EdgeInsets.only(top: 8),
                          child: Text("⚠️ En TLS ON debes escanear el QR para token + fp."),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 12),

          Card(
            child: Column(
              children: [
                SwitchListTile(
                  title: const Text("Usar usuario/contraseña"),
                  subtitle: const Text("Solo con TLS ON (si el daemon lo activó)"),
                  value: _useUserPass,
                  onChanged: _useTls ? (v) => setState(() => _useUserPass = v) : null,
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                  child: Column(
                    children: [
                      TextField(
                        controller: _userCtrl,
                        enabled: _useTls && _useUserPass,
                        decoration: const InputDecoration(
                          labelText: "Usuario",
                          border: OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 10),
                      TextField(
                        controller: _passCtrl,
                        enabled: _useTls && _useUserPass,
                        obscureText: true,
                        decoration: const InputDecoration(
                          labelText: "Contraseña",
                          border: OutlineInputBorder(),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 12),

          Card(
            child: SwitchListTile(
              title: const Text("Cifrado de payload (legacy)"),
              subtitle: const Text("AES-GCM sobre WS (opcional si ya usas TLS)"),
              value: _securePayload,
              onChanged: (v) => setState(() => _securePayload = v),
            ),
          ),
        ],
      ),
    );
  }
}
