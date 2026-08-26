import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'auth_interceptor.dart';
import 'refresh_interceptor.dart';
import '../storage/token_storage.dart';

/// Base URL for the API (`/api/v1` — contracts/api.md). Overridable for
/// devices/simulators via:
/// `flutter run --dart-define=API_BASE_URL=http://<host>:8080/api/v1`
const String apiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://10.0.2.2:8080/api/v1', // Android emulator → host loopback
);

/// Configured Dio client: Bearer injection, single-flight token refresh, and
/// debug logging.
class ApiClient {
  ApiClient({
    required TokenStorage storage,
    required void Function() onSessionExpired,
  }) : dio = Dio(
          BaseOptions(
            baseUrl: apiBaseUrl,
            connectTimeout: const Duration(seconds: 15),
            receiveTimeout: const Duration(seconds: 20),
            headers: {'Content-Type': 'application/json; charset=utf-8'},
            validateStatus: (code) => code != null && code >= 200 && code < 400,
          ),
        ) {
    dio.interceptors.addAll([
      AuthInterceptor(storage),
      RefreshInterceptor(dio: dio, storage: storage, onSessionExpired: onSessionExpired),
      if (kDebugMode)
        LogInterceptor(requestBody: true, responseBody: true, error: true),
    ]);
  }

  /// Shared entrypoint for all feature repositories.
  final Dio dio;
}
