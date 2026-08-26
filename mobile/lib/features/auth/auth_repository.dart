import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/api_client.dart';
import '../../core/network/api_exception.dart';
import 'auth_controller.dart';
import 'models/user_session.dart';

/// Shared Dio client wired to the auth session: 401s trigger the rotating
/// refresh flow; a dead refresh family drops the user back to login.
final apiClientProvider = Provider<ApiClient>((ref) {
  final client = ApiClient(
    storage: ref.watch(tokenStorageProvider),
    onSessionExpired: () =>
        ref.read(authControllerProvider.notifier).forceLogout(),
  );
  ref.onDispose(() => client.dio.close());
  return client;
});

final authRepositoryProvider = Provider<AuthRepository>(
  (ref) => AuthRepository(ref.watch(apiClientProvider).dio),
);

/// Verified session material returned by [AuthRepository.verifyOtp]; the
/// screen hands it to [AuthController.completeLogin] for persistence.
class VerifyResult {
  const VerifyResult({
    required this.user,
    required this.accessToken,
    required this.refreshToken,
  });

  final UserSession user;
  final String accessToken;
  final String refreshToken;
}

/// US1 auth endpoints (contracts/api.md "Auth P0-10").
class AuthRepository {
  AuthRepository(this._dio);

  final Dio _dio;

  /// `POST /auth/otp/request`. Returns the dev-mode code (dev deployments
  /// only — production answers 204 with no body).
  Future<String?> requestOtp(String phone) async {
    final res = await _dio.post<dynamic>(
      '/auth/otp/request',
      data: {'phone': phone},
    );
    final data = res.data;
    if (data is Map && data['dev_code'] != null) {
      return data['dev_code'].toString();
    }
    return null;
  }

  /// `POST /auth/otp/verify` → token pair + user context.
  Future<VerifyResult> verifyOtp({
    required String phone,
    required String code,
  }) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/auth/otp/verify',
      data: {'phone': phone, 'code': code},
    );
    final data = res.data;
    final access = data?['access_token']?.toString();
    final refresh = data?['refresh_token']?.toString();
    final userJson = data?['user'];
    if (access == null || refresh == null || userJson is! Map) {
      throw ApiException(code: 'unknown');
    }

    return VerifyResult(
      user: UserSession.fromJson(Map<String, dynamic>.from(userJson)),
      accessToken: access,
      refreshToken: refresh,
    );
  }
}
