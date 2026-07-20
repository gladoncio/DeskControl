class ConnectionData {
  final bool autoDiscover;

  /// Manual
  final String host;
  final int port;

  /// TLS (WSS)
  final bool useTls;

  /// Token (obligatorio si TLS está ON en tu política)
  final String token;

  /// Fingerprint SHA-256 del certificado TLS (HEX) (pinning)
  final String certFpSha256Hex;

  /// Cuenta opcional (solo si el daemon la activó)
  final bool useUserPass;
  final String username;
  final String password;

  /// (Legacy) cifrado de payload AES-GCM (puedes dejarlo OFF si usas TLS)
  final bool securePayload;

  const ConnectionData({
    required this.autoDiscover,
    required this.host,
    required this.port,
    required this.useTls,
    required this.token,
    required this.certFpSha256Hex,
    required this.useUserPass,
    required this.username,
    required this.password,
    required this.securePayload,
  });

  ConnectionData copyWith({
    bool? autoDiscover,
    String? host,
    int? port,
    bool? useTls,
    String? token,
    String? certFpSha256Hex,
    bool? useUserPass,
    String? username,
    String? password,
    bool? securePayload,
  }) {
    return ConnectionData(
      autoDiscover: autoDiscover ?? this.autoDiscover,
      host: host ?? this.host,
      port: port ?? this.port,
      useTls: useTls ?? this.useTls,
      token: token ?? this.token,
      certFpSha256Hex: certFpSha256Hex ?? this.certFpSha256Hex,
      useUserPass: useUserPass ?? this.useUserPass,
      username: username ?? this.username,
      password: password ?? this.password,
      securePayload: securePayload ?? this.securePayload,
    );
  }
}
