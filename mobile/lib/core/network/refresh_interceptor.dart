import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';


import '../storage/token_storage.dart';
import 'api_exception.dart';

/// Single-flight refresh flow: on the first 401 it exchanges the refresh token
/// at `POST /auth/refresh`, persists the rotated pair, and replays the original
/// request. If refresh fails the session is cleared and [onSessionExpired]
/// fires so the router can drop back to login.
///
/// Extends [QueuedInterceptor] so concurrent 401s wait instead of triggering
/// parallel refreshes.
class RefreshInterceptor extends QueuedInterceptor {
  RefreshInterceptor({
    required Dio dio,
    required TokenStorage storage,
    required VoidCallback onSessionExpired,
  })  : _dio = dio,
        _bareDio = Dio(BaseOptions(baseUrl: dio.options.baseUrl)),
        _storage = storage,
        _onSessionExpired = onSessionExpired;

  static const _authPrefixes = ['/auth/refresh', '/auth/otp/'];

  final Dio _dio;

  /// Refresh calls must bypass this interceptor chain entirely.
  final Dio _bareDio;
  final TokenStorage _storage;
  final VoidCallback _onSessionExpired;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    if (!_shouldAttempt(err)) return handler.next(err);

    try {
      final session = await _storage.read();
      final refreshToken = session?.refreshToken;
      if (refreshToken == null) return handler.next(err);

      final response = await _bareDio.post<Map<String, dynamic>>(
        '/auth/refresh',
        data: {'refresh_token': refreshToken},
      );
      final data = response.data;
      final newAccess = data?['access_token']?.toString();
      final newRefresh = data?['refresh_token']?.toString();
      if (newAccess == null || newRefresh == null) {
        throw ApiException(code: 'UNAUTHENTICATED');
      }

      await _storage.save(StoredSession(
        accessToken: newAccess,
        refreshToken: newRefresh,
        userJson: session!.userJson,
      ));

      // Replay the original request with the fresh access token.
      final opts = err.requestOptions..headers['Authorization'] = 'Bearer $newAccess';
      final replayed = await _dio.fetch<dynamic>(opts);
      return handler.resolve(replayed);
    } on Exception catch (_) {
      await _storage.clear();
      _onSessionExpired();
      return handler.next(err);
    }
  }

  bool _shouldAttempt(DioException err) {
    if (err.response?.statusCode != 401) return false;
    final path = err.requestOptions.path;
    return !_authPrefixes.any(path.startsWith);
  }
}
