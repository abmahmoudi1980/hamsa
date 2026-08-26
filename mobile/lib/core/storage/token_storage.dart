import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Session material persisted in Android Keystore-backed secure storage.
class StoredSession {
  const StoredSession({
    required this.accessToken,
    required this.refreshToken,
    required this.userJson,
  });

  final String accessToken;
  final String refreshToken;
  final String userJson;
}

/// Thin wrapper around [FlutterSecureStorage] for the rotating token pair.
///
/// Under `flutter test` there is no platform channel; the plugin's futures
/// never complete there, so storage calls short-circuit (no session) instead
/// of hanging the widget-test event loop.
class TokenStorage {
  static const _accessKey = 'hamsa.access_token';
  static const _refreshKey = 'hamsa.refresh_token';
  static const _userKey = 'hamsa.user';

  static const FlutterSecureStorage _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );

  static final bool _inTests =
      !kIsWeb && Platform.environment.containsKey('FLUTTER_TEST');

  Future<void> save(StoredSession session) async {
    if (_inTests) return;
    await _storage.write(key: _accessKey, value: session.accessToken);
    await _storage.write(key: _refreshKey, value: session.refreshToken);
    await _storage.write(key: _userKey, value: session.userJson);
  }

  /// Returns the stored session or null when absent/unavailable.
  Future<StoredSession?> read() async {
    if (_inTests) return null;
    try {
      final access = await _storage.read(key: _accessKey);
      final refresh = await _storage.read(key: _refreshKey);
      final user = await _storage.read(key: _userKey);
      if (access == null || refresh == null || user == null) return null;
      return StoredSession(
        accessToken: access,
        refreshToken: refresh,
        userJson: user,
      );
    } on Exception catch (e) {
      if (kDebugMode) debugPrint('TokenStorage.read failed: $e');
      return null;
    }
  }

  Future<void> clear() async {
    if (_inTests) return;
    try {
      await _storage.delete(key: _accessKey);
      await _storage.delete(key: _refreshKey);
      await _storage.delete(key: _userKey);
    } on Exception catch (e) {
      if (kDebugMode) debugPrint('TokenStorage.clear failed: $e');
    }
  }
}
