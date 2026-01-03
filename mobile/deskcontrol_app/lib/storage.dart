import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';

import 'connection_config.dart';

class AppStorage {
  // -------- Config keys ----------
  static const _kSensitivity = 'cfg_sensitivity';
  static const _kScrollSpeed = 'cfg_scroll_speed';
  static const _kThemeMode = 'cfg_theme_mode';
  static const _kHoldDelayMs = 'cfg_hold_delay_ms';

  // -------- Connection keys ----------
  static const _kConnAutoDiscover = 'conn_auto_discover';
  static const _kConnHost = 'conn_host';
  static const _kConnPort = 'conn_port';

  static const _kConnUseTls = 'conn_use_tls';
  static const _kConnToken = 'conn_token';
  static const _kConnCertFp = 'conn_cert_fp_sha256_hex';

  static const _kConnUseUserPass = 'conn_use_userpass';
  static const _kConnUsername = 'conn_username';
  static const _kConnPassword = 'conn_password'; // ⚠️ ideal: secure storage

  static const _kConnSecurePayload = 'conn_secure_payload';

  // -------- Keyboard keys ----------
  static const _kKeys = 'keys_v1';
  static const _kCombos = 'combos_v1';
  static const _kVisibleKeys = 'keys_visible_v1';

  // ---------- Connection ----------
  static Future<ConnectionData> loadConnection() async {
    final sp = await SharedPreferences.getInstance();
    return ConnectionData(
      autoDiscover: sp.getBool(_kConnAutoDiscover) ?? true,
      host: sp.getString(_kConnHost) ?? "",
      port: sp.getInt(_kConnPort) ?? 54545,

      useTls: sp.getBool(_kConnUseTls) ?? false,
      token: sp.getString(_kConnToken) ?? "",
      certFpSha256Hex: sp.getString(_kConnCertFp) ?? "",

      useUserPass: sp.getBool(_kConnUseUserPass) ?? false,
      username: sp.getString(_kConnUsername) ?? "",
      password: sp.getString(_kConnPassword) ?? "",

      securePayload: sp.getBool(_kConnSecurePayload) ?? false,
    );
  }

  static Future<void> saveConnection(ConnectionData c) async {
    final sp = await SharedPreferences.getInstance();

    await sp.setBool(_kConnAutoDiscover, c.autoDiscover);
    await sp.setString(_kConnHost, c.host);
    await sp.setInt(_kConnPort, c.port);

    await sp.setBool(_kConnUseTls, c.useTls);
    await sp.setString(_kConnToken, c.token);
    await sp.setString(_kConnCertFp, c.certFpSha256Hex);

    await sp.setBool(_kConnUseUserPass, c.useUserPass);
    await sp.setString(_kConnUsername, c.username);
    await sp.setString(_kConnPassword, c.password);

    await sp.setBool(_kConnSecurePayload, c.securePayload);
  }

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

  Map<String, dynamic> toJson() => {'vk': vk, 'scan': scan, 'ext': ext};

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
  final String? keyName;

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

  Map<String, dynamic> toJson() =>
      {'id': id, 'name': name, 'keys': keys, 'tapIndex': tapIndex};

  static ComboItem fromJson(Map<String, dynamic> m) {
    final keys = (m['keys'] is List)
        ? (m['keys'] as List).map((e) => e.toString()).toList()
        : <String>[];
    final tapIndex = (m['tapIndex'] as num?)?.toInt() ?? 0;
    return ComboItem(
      id: (m['id'] ?? '').toString(),
      name: (m['name'] ?? '').toString(),
      keys: keys,
      tapIndex: tapIndex,
    );
  }
}
