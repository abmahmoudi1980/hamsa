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

/// Session material returned by the login/register/setup endpoints; the
/// screen hands it to [AuthController.completeLogin] for persistence.
class AuthResult {
  const AuthResult({
    required this.user,
    required this.accessToken,
    required this.refreshToken,
  });

  final UserSession user;
  final String accessToken;
  final String refreshToken;
}

/// One issued invite (`POST /auth/invites` — 002-multi-manager-support):
/// the code plus the role it grants (`resident` | `manager`) and its
/// validity window in days.
class InviteResult {
  const InviteResult({
    required this.code,
    required this.role,
    required this.expiresInDays,
  });

  final String code;
  final String role;
  final int expiresInDays;
}

/// Auth endpoints (phone + password; contracts/api.md "Auth P0-10").
class AuthRepository {
  AuthRepository(this._dio);

  final Dio _dio;

  /// `POST /auth/login` → token pair + user context.
  Future<AuthResult> login({
    required String phone,
    required String password,
  }) async {
    return _session('/auth/login', {'phone': phone, 'password': password});
  }

  /// `POST /auth/setup` — first-account bootstrap (fresh deployments only).
  Future<AuthResult> setup({
    required String phone,
    required String password,
    String? name,
  }) async {
    return _session('/auth/setup', {
      'phone': phone,
      'password': password,
      if (name != null && name.isNotEmpty) 'name': name,
    });
  }

  /// `POST /auth/register` — redeem a one-time invite code. Registers a new
  /// resident or resets an existing user's password (recovery path).
  Future<AuthResult> register({
    required String phone,
    required String code,
    required String password,
    String? name,
  }) async {
    return _session('/auth/register', {
      'phone': phone,
      'code': code,
      'password': password,
      if (name != null && name.isNotEmpty) 'name': name,
    });
  }

  /// `POST /auth/password` — change the signed-in user's password (204).
  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  }) async {
    await _dio.post<void>('/auth/password', data: {
      'current_password': currentPassword,
      'new_password': newPassword,
    });
  }

  /// `POST /auth/invites` (manager or superadmin) → one-time invite code
  /// carrying [role] (`resident` | `manager`; contracts/api.md 002 delta).
  Future<InviteResult> createInvite(String phone, String role) async {
    final res = await _dio.post<Map<String, dynamic>>(
      '/auth/invites',
      data: {'phone': phone, 'role': role},
    );
    final code = res.data?['code']?.toString();
    if (code == null || code.isEmpty) {
      throw ApiException(code: 'unknown');
    }
    return InviteResult(
      code: code,
      role: res.data?['role']?.toString() ?? role,
      expiresInDays: (res.data?['expires_in_days'] as num?)?.toInt() ?? 7,
    );
  }

  /// Shared session-body parsing for login/register/setup.
  Future<AuthResult> _session(String path, Map<String, dynamic> data) async {
    final res = await _dio.post<Map<String, dynamic>>(path, data: data);
    final access = res.data?['access_token']?.toString();
    final refresh = res.data?['refresh_token']?.toString();
    final userJson = res.data?['user'];
    if (access == null || refresh == null || userJson is! Map) {
      throw ApiException(code: 'unknown');
    }

    return AuthResult(
      user: UserSession.fromJson(Map<String, dynamic>.from(userJson)),
      accessToken: access,
      refreshToken: refresh,
    );
  }
}
