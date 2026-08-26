import 'package:dio/dio.dart';

import '../storage/token_storage.dart';

/// Adds `Authorization: Bearer <access_token>` to every outgoing request
/// when a session exists (contracts/api.md).
class AuthInterceptor extends Interceptor {
  AuthInterceptor(this._storage);

  final TokenStorage _storage;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    final session = await _storage.read();
    if (session != null) {
      options.headers['Authorization'] = 'Bearer ${session.accessToken}';
    }
    handler.next(options);
  }
}
