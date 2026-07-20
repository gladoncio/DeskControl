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
  static const _kConnProfileId = 'conn_last_profile_id';

  static const _kConnUseTls = 'conn_use_tls';
  static const _kConnToken = 'conn_token';
  static const _kConnCertFp = 'conn_cert_fp_sha256_hex';

  static const _kConnUseUserPass = 'conn_use_userpass';
  static const _kConnUsername = 'conn_username';
  static const _kConnPassword = 'conn_password';

  static const _kConnSecurePayload = 'conn_secure_payload';

  // -------- Perfil prefix ----------
  static String _profilePrefix(String profileId) => 'profile:$profileId:';

  // -------- Connection history ----------
  static const _kConnHistory = 'conn_history_v2';

  static Future<List<ConnectionRecord>> loadConnectionHistory() async {
    final sp = await SharedPreferences.getInstance();
    final raw = sp.getString(_kConnHistory);
    if (raw == null || raw.trim().isEmpty) return [];
    try {
      final arr = (jsonDecode(raw) as List).cast<dynamic>();
      return arr
          .map((e) => ConnectionRecord.fromJson(
              (e as Map).cast<String, dynamic>()))
          .toList();
    } catch (_) {
      return [];
    }
  }

  static Future<void> saveConnectionHistory(List<ConnectionRecord> records) async {
    final sp = await SharedPreferences.getInstance();
    final arr = records.map((e) => e.toJson()).toList();
    await sp.setString(_kConnHistory, jsonEncode(arr));
  }

  static Future<void> addConnectionRecord(ConnectionRecord r) async {
    final records = await loadConnectionHistory();
    records.removeWhere((x) => x.profileId == r.profileId);
    records.insert(0, r);
    if (records.length > 20) records.removeRange(20, records.length);
    await saveConnectionHistory(records);
  }

  static Future<void> removeConnectionRecord(String profileId) async {
    final records = await loadConnectionHistory();
    records.removeWhere((x) => x.profileId == profileId);
    await saveConnectionHistory(records);
  }

  // -------- Profile name (custom alias) ----------
  static const _kProfileNamePrefix = 'profile_name:';
  static const _kDeviceNamePrefix = 'device_name:';

  /// Load the custom name for a device or profile.
  /// If [deviceId] is non-empty and has a custom name, returns it.
  /// Otherwise falls back to profileId-based name.
  static Future<String> loadProfileName(String profileId, {String deviceId = ''}) async {
    final sp = await SharedPreferences.getInstance();
    if (deviceId.isNotEmpty) {
      final deviceName = sp.getString('$_kDeviceNamePrefix$deviceId');
      if (deviceName != null && deviceName.isNotEmpty) return deviceName;
    }
    return sp.getString('$_kProfileNamePrefix$profileId') ?? '';
  }

  /// Save a custom name for a profile.
  /// Pass [deviceId] to save a device-scoped alias (shared across IPs).
  static Future<void> saveProfileName(String profileId, String name, {String deviceId = ''}) async {
    final sp = await SharedPreferences.getInstance();
    if (deviceId.isNotEmpty) {
      if (name.trim().isEmpty) {
        await sp.remove('$_kDeviceNamePrefix$deviceId');
      } else {
        await sp.setString('$_kDeviceNamePrefix$deviceId', name.trim());
      }
    } else {
      if (name.trim().isEmpty) {
        await sp.remove('$_kProfileNamePrefix$profileId');
      } else {
        await sp.setString('$_kProfileNamePrefix$profileId', name.trim());
      }
    }
  }

  // -------- Export/Import profile ----------
  static Future<String?> exportProfile(String profileId) async {
    final keys = await loadKeys(profileId: profileId);
    final combos = await loadCombos(profileId: profileId);
    final visible = await loadVisibleKeyIds(profileId: profileId);
    if (keys.isEmpty && combos.isEmpty) return null;
    return jsonEncode({
      'version': 1,
      'profileId': profileId,
      'keys': keys.map((e) => e.toJson()).toList(),
      'combos': combos.map((e) => e.toJson()).toList(),
      'visibleKeyIds': visible,
    });
  }

  static Future<void> importProfile(String profileId, String jsonStr) async {
    final obj = jsonDecode(jsonStr);
    if (obj is! Map) return;
    final keys = (obj['keys'] as List?)
            ?.map((e) =>
                KeyItem.fromJson((e as Map).cast<String, dynamic>()))
            .toList() ??
        [];
    final combos = (obj['combos'] as List?)
            ?.map((e) =>
                ComboItem.fromJson((e as Map).cast<String, dynamic>()))
            .toList() ??
        [];
    final visible = (obj['visibleKeyIds'] as List?)
            ?.map((e) => e.toString())
            .toList() ??
        <String>[];
    await saveKeys(keys, profileId: profileId);
    await saveCombos(combos, profileId: profileId);
    await saveVisibleKeyIds(visible, profileId: profileId);
  }

  // -------- Keyboard keys (por perfil) ----------
  static String _kKeys(String profileId) =>
      '${_profilePrefix(profileId)}keys_v1';
  static String _kCombos(String profileId) =>
      '${_profilePrefix(profileId)}combos_v1';
  static String _kVisibleKeys(String profileId) =>
      '${_profilePrefix(profileId)}keys_visible_v1';

  // ---------- Conexión ----------
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

  // ---------- Perfil activo ----------
  static Future<String> loadLastProfileId() async {
    final sp = await SharedPreferences.getInstance();
    return sp.getString(_kConnProfileId) ?? "";
  }

  static Future<void> saveLastProfileId(String id) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setString(_kConnProfileId, id);
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

  // -------- Keys (por perfil) ----------
  static Future<void> saveKeys(List<KeyItem> keys, {String profileId = ""}) async {
    final sp = await SharedPreferences.getInstance();
    final arr = keys.map((e) => e.toJson()).toList();
    await sp.setString(_kKeys(profileId), jsonEncode(arr));
  }

  static Future<List<KeyItem>> loadKeys({String profileId = ""}) async {
    final sp = await SharedPreferences.getInstance();
    final raw = sp.getString(_kKeys(profileId));
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

  // -------- Combos (por perfil) ----------
  static Future<void> saveCombos(List<ComboItem> combos, {String profileId = ""}) async {
    final sp = await SharedPreferences.getInstance();
    final arr = combos.map((e) => e.toJson()).toList();
    await sp.setString(_kCombos(profileId), jsonEncode(arr));
  }

  static Future<List<ComboItem>> loadCombos({String profileId = ""}) async {
    final sp = await SharedPreferences.getInstance();
    final raw = sp.getString(_kCombos(profileId));
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

  // -------- Visible keys (por perfil) ----------
  static Future<void> saveVisibleKeyIds(List<String> ids, {String profileId = ""}) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setStringList(_kVisibleKeys(profileId), ids);
  }

  static Future<List<String>> loadVisibleKeyIds({String profileId = ""}) async {
    final sp = await SharedPreferences.getInstance();
    return sp.getStringList(_kVisibleKeys(profileId)) ?? <String>[];
  }
}

class ConnectionRecord {
  final String profileId;
  final String host;
  final int port;
  final String name;
  final String deviceId;
  final bool useTls;
  final int lastConnected; // unix timestamp ms

  const ConnectionRecord({
    required this.profileId,
    required this.host,
    required this.port,
    required this.name,
    this.deviceId = '',
    required this.useTls,
    required this.lastConnected,
  });

  Map<String, dynamic> toJson() => {
        'profileId': profileId,
        'host': host,
        'port': port,
        'name': name,
        'deviceId': deviceId,
        'useTls': useTls,
        'lastConnected': lastConnected,
      };

  static ConnectionRecord fromJson(Map<String, dynamic> m) =>
      ConnectionRecord(
        profileId: (m['profileId'] ?? '').toString(),
        host: (m['host'] ?? '').toString(),
        port: (m['port'] as num?)?.toInt() ?? 54545,
        name: (m['name'] ?? '').toString(),
        deviceId: (m['deviceId'] ?? '').toString(),
        useTls: (m['useTls'] == true),
        lastConnected: (m['lastConnected'] as num?)?.toInt() ?? 0,
      );
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
