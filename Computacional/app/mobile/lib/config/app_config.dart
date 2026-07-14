// lib/config/app_config.dart
//
// Compile-time configuration via --dart-define. Absent flags fall back to the
// production default. Examples:
//   Android emulator: --dart-define=API_BASE_URL=http://10.0.2.2:8080
//   Physical device:  --dart-define=API_BASE_URL=http://<LAN-IP>:8080

/// Centralised compile-time configuration. Pure config — never import dart:io
/// or check Platform here.
abstract final class AppConfig {
  // API base URL. Trailing slash omitted so paths read as '$apiBaseUrl/api/...'.
  // Default: production gateway.
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://3.22.171.3:8080',
  );

  // Prevent instantiation — this is a pure namespace.
  AppConfig._();
}
