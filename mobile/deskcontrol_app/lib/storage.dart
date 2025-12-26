import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';

class AppStorage {
  // -------- Config keys ----------
  static const _kSensitivity = 'cfg_sensitivity';
  static const _kScrollSpeed = 'cfg_scroll_speed';
  static const _kThemeMode = 'cfg_theme_mode';

  // ✅ NEW: tiempo (ms) para activar click mantenido (long press)
  static const _kHoldDelayMs = 'cfg_hold_delay_ms';

  // -------- Keyboard keys ----------
  static const _kKeys = 'keys_v1';
  static const _kCombos = 'combos_v1';

  // ✅ NEW: qué teclas se muestran en la vista User
  static const _kVisibleKeys = 'keys_visible_v1';

  // ---------- Config ----------
  static Future<void> saveConfig({
    required double sensitivity,
    required double scrollSpeed,
    required String themeMode,
    required int holdDelayMs,
  }) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setDouble(_kSensitivity, sensitivity);
    await sp.setDouble(_kScrollSpeed, scrollSpeed);
    await sp.setString(_kThemeMode, themeMode);
    await sp.setInt(_kHoldDelayMs, holdDelayMs);
  }

  static Future<ConfigData> loadConfig() async {
    final sp = await SharedPreferences.getInstance();
    return ConfigData(
      sensitivity: sp.getDouble(_kSensitivity) ?? 1.0,
      scrollSpeed: sp.getDouble(_kScrollSpeed) ?? 1.0,
      themeMode: sp.getString(_kThemeMode) ?? "system",
      holdDelayMs: sp.getInt(_kHoldDelayMs) ?? 350,
    );
  }

  // -------- Visible keys (User screen) ----------
  static Future<void> saveVisibleKeyIds(List<String> ids) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setStringList(_kVisibleKeys, ids);
  }

  static Future<List<String>> loadVisibleKeyIds() async {
    final sp = await SharedPreferences.getInstance();
    return sp.getStringList(_kVisibleKeys) ?? <String>[];
  }

  // -------- Keys ----------
  static Future<void> saveKeys(List<KeyItem> keys) async {
    final sp = await SharedPreferences.getInstance();
    final arr = keys.map((e) => e.toJson()).toList();
    await sp.setString(_kKeys, jsonEncode(arr));
  }

  static Future<List<KeyItem>> loadKeys() async {
    final sp = await SharedPreferences.getInstance();
    final raw = sp.getString(_kKeys);
    if (raw == null || raw.trim().isEmpty) return <KeyItem>[];
    try {
      final arr = (jsonDecode(raw) as List).cast<dynamic>();
      return arr
          .map((e) => KeyItem.fromJson((e as Map).cast<String, dynamic>()))
          .toList();
    } catch (_) {
      return <KeyItem>[];
    }
  }

  // -------- Combos ----------
  static Future<void> saveCombos(List<ComboItem> combos) async {
    final sp = await SharedPreferences.getInstance();
    final arr = combos.map((e) => e.toJson()).toList();
    await sp.setString(_kCombos, jsonEncode(arr));
  }

  static Future<List<ComboItem>> loadCombos() async {
    final sp = await SharedPreferences.getInstance();
    final raw = sp.getString(_kCombos);
    if (raw == null || raw.trim().isEmpty) return <ComboItem>[];
    try {
      final arr = (jsonDecode(raw) as List).cast<dynamic>();
      return arr
          .map((e) => ComboItem.fromJson((e as Map).cast<String, dynamic>()))
          .toList();
    } catch (_) {
      return <ComboItem>[];
    }
  }
}

class ConfigData {
  final double sensitivity;
  final double scrollSpeed;
  final String themeMode;

  // ✅ NEW
  final int holdDelayMs;

  const ConfigData({
    required this.sensitivity,
    required this.scrollSpeed,
    required this.themeMode,
    required this.holdDelayMs,
  });
}

class KeySpec {
  final int vk;
  final int scan;
  final bool ext;

  const KeySpec({required this.vk, required this.scan, required this.ext});

  Map<String, dynamic> toJson() => {
        'vk': vk,
        'scan': scan,
        'ext': ext,
      };

  static KeySpec fromJson(Map<String, dynamic> m) => KeySpec(
        vk: (m['vk'] as num?)?.toInt() ?? 0,
        scan: (m['scan'] as num?)?.toInt() ?? 0,
        ext: (m['ext'] == true),
      );
}

class KeyItem {
  final String id;
  final String name;
  final bool useVK;
  final KeySpec? keySpec;
  final String? keyName; // si no es VK, texto a enviar

  const KeyItem({
    required this.id,
    required this.name,
    required this.useVK,
    required this.keySpec,
    required this.keyName,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'useVK': useVK,
        'keySpec': keySpec?.toJson(),
        'keyName': keyName,
      };

  static KeyItem fromJson(Map<String, dynamic> m) => KeyItem(
        id: (m['id'] ?? '').toString(),
        name: (m['name'] ?? '').toString(),
        useVK: (m['useVK'] == true),
        keySpec: (m['keySpec'] is Map)
            ? KeySpec.fromJson((m['keySpec'] as Map).cast<String, dynamic>())
            : null,
        keyName: m['keyName']?.toString(),
      );
}

class ComboItem {
  final String id;
  final String name;
  final List<String> keys;
  final int tapIndex;

  const ComboItem({
    required this.id,
    required this.name,
    required this.keys,
    required this.tapIndex,
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'keys': keys,
        'tapIndex': tapIndex,
      };

  static ComboItem fromJson(Map<String, dynamic> m) {
    final keys = (m['keys'] is List)
        ? (m['keys'] as List).map((e) => e.toString()).toList()
        : <String>[];
    final tapIndex = (m['tapIndex'] as num?)?.toInt() ??
        (keys.isEmpty ? 0 : keys.length - 1);
    return ComboItem(
      id: (m['id'] ?? '').toString(),
      name: (m['name'] ?? '').toString(),
      keys: keys,
      tapIndex: tapIndex.clamp(0, keys.isEmpty ? 0 : keys.length - 1),
    );
  }
}
