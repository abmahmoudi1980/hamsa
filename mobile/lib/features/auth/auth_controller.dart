import 'dart:async';
import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/storage/token_storage.dart';
import 'models/user_session.dart';

enum AuthStatus { unknown, authenticated, unauthenticated }

class AuthState {
  const AuthState({required this.status, this.user});

  const AuthState.unknown() : this(status: AuthStatus.unknown);
  const AuthState.authenticated(UserSession user)
      : this(status: AuthStatus.authenticated, user: user);
  const AuthState.unauthenticated() : this(status: AuthStatus.unauthenticated);

  final AuthStatus status;
  final UserSession? user;
}

final tokenStorageProvider = Provider<TokenStorage>((ref) => TokenStorage());

/// Restores the persisted session at startup and owns login/logout state
/// transitions. The router listens to this provider and re-evaluates its
/// redirect on every change.
class AuthController extends Notifier<AuthState> {
  @override
  AuthState build() {
    scheduleMicrotask(_restore);
    return const AuthState.unknown();
  }

  Future<void> _restore() async {
    final session = await ref.read(tokenStorageProvider).read();
    if (session == null) {
      state = const AuthState.unauthenticated();
      return;
    }
    try {
      state = AuthState.authenticated(
        UserSession.fromJson(jsonDecode(session.userJson) as Map<String, dynamic>),
      );
    } on FormatException {
      await _clearAndSignOut();
    }
  }

  /// Persists a successful login (tokens already validated by the caller —
  /// real OTP screens arrive in US1/T026) and flips to authenticated.
  Future<void> completeLogin({
    required UserSession user,
    required String accessToken,
    required String refreshToken,
  }) async {
    await ref.read(tokenStorageProvider).save(StoredSession(
          accessToken: accessToken,
          refreshToken: refreshToken,
          userJson: jsonEncode(user.toJson()),
        ));
    state = AuthState.authenticated(user);
  }

  /// User-initiated logout.
  Future<void> logout() => _clearAndSignOut();

  /// Session-expiry path (refresh reuse detected by RefreshInterceptor).
  Future<void> forceLogout() => _clearAndSignOut();

  Future<void> _clearAndSignOut() async {
    await ref.read(tokenStorageProvider).clear();
    state = const AuthState.unauthenticated();
  }
}

final authControllerProvider =
    NotifierProvider<AuthController, AuthState>(AuthController.new);
