import 'package:flutter/material.dart';
import '../desk_socket.dart';

class AppInfo {
  final int hwnd;
  final int pid;
  final String title;
  final String? exe;
  final bool minimized;

  const AppInfo({
    required this.hwnd,
    required this.pid,
    required this.title,
    this.exe,
    required this.minimized,
  });

  static AppInfo fromJson(Map<String, dynamic> m) {
    return AppInfo(
      hwnd: (m['hwnd'] as num?)?.toInt() ?? 0,
      pid: (m['pid'] as num?)?.toInt() ?? 0,
      title: (m['title'] ?? '').toString(),
      exe: m['exe']?.toString(),
      minimized: (m['minimized'] == true),
    );
  }
}

class AppsTab extends StatefulWidget {
  final DeskSocket desk;
  const AppsTab({super.key, required this.desk});

  @override
  State<AppsTab> createState() => _AppsTabState();
}

class _AppsTabState extends State<AppsTab> {
  bool _loading = false;
  String? _error;
  List<AppInfo> _apps = [];

  String _baseName(String? path) {
    if (path == null || path.isEmpty) return '';
    final p = path.replaceAll('\\', '/');
    final parts = p.split('/');
    return parts.isEmpty ? path : parts.last;
  }

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final res = await widget.desk.request(
        'apps_list',
        timeout: const Duration(seconds: 5),
      );

      if (res['type'] == 'error') {
        setState(() => _error = res['error']?.toString() ?? 'error');
        return;
      }

      final appsRaw = res['apps'];
      if (appsRaw is! List) {
        setState(() => _error = 'Respuesta inválida');
        return;
      }

      final apps = appsRaw
          .whereType<Map>()
          .map((e) => AppInfo.fromJson(Map<String, dynamic>.from(e)))
          .where((a) => a.hwnd != 0 && a.title.trim().isNotEmpty)
          .toList();

      apps.sort((a, b) => a.title.toLowerCase().compareTo(b.title.toLowerCase()));

      setState(() => _apps = apps);
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _action(AppInfo a, String action) async {
    try {
      await widget.desk.request(
        'app_action',
        payload: {
          'hwnd': a.hwnd,
          'action': action,
        },
        timeout: const Duration(seconds: 3),
      );
      // refresca estado (minimized / etc)
      _refresh();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Error: $e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading && _apps.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null && _apps.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Error: $_error', textAlign: TextAlign.center),
              const SizedBox(height: 12),
              ElevatedButton.icon(
                onPressed: _refresh,
                icon: const Icon(Icons.refresh),
                label: const Text('Reintentar'),
              )
            ],
          ),
        ),
      );
    }

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              const Expanded(
                child: Text(
                  'Apps (ventanas)',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
                ),
              ),
              IconButton(
                onPressed: _refresh,
                icon: const Icon(Icons.refresh),
                tooltip: 'Actualizar',
              ),
            ],
          ),
        ),
        const Divider(height: 1),
        Expanded(
          child: _apps.isEmpty
              ? const Center(child: Text('Sin apps'))
              : ListView.separated(
                  itemCount: _apps.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (ctx, i) {
                    final a = _apps[i];
                    return ListTile(
                      leading: Icon(a.minimized ? Icons.minimize : Icons.apps),
                      title: Text(a.title),
                      subtitle: Text(
                        'PID ${a.pid} • ${a.minimized ? 'Minimizada' : 'Visible'}'
                        '${a.exe != null ? ' • ${_baseName(a.exe)}' : ''}',
                      ),
                      trailing: PopupMenuButton<String>(
                        onSelected: (v) => _action(a, v),
                        itemBuilder: (_) => [
                          PopupMenuItem(
                            value: a.minimized ? 'restore' : 'minimize',
                            child: Text(a.minimized ? 'Restaurar' : 'Minimizar'),
                          ),
                          const PopupMenuItem(
                            value: 'maximize',
                            child: Text('Maximizar'),
                          ),
                          const PopupMenuItem(
                            value: 'activate',
                            child: Text('Activar'),
                          ),
                          const PopupMenuItem(
                            value: 'close',
                            child: Text('Cerrar'),
                          ),
                        ],
                      ),
                      onTap: () => _action(a, 'activate'),
                    );
                  },
                ),
        ),
      ],
    );
  }
}
