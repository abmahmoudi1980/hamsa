import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/auth_controller.dart';
import '../../features/auth/models/user_session.dart';
import '../../features/auth/screens/login_screen.dart';
import '../../features/auth/screens/otp_screen.dart';
import '../../features/dashboard/screens/manager_shell_screen.dart';
import '../../features/home/screens/resident_shell_screen.dart';
import 'splash_screen.dart';

/// Role-based root routing (plan.md structure): login ↔ manager shell ↔
/// resident shell, with auth-state redirect re-evaluated on every
/// [authControllerProvider] change.
final routerProvider = Provider<GoRouter>((ref) {
  final router = GoRouter(
    initialLocation: '/',
    redirect: (context, state) => _redirect(ref, state),
    routes: [
      GoRoute(path: '/', builder: (_, _) => const SplashScreen()),
      GoRoute(path: '/login', builder: (_, _) => const LoginScreen()),
      GoRoute(
        path: '/login/otp',
        builder: (_, state) =>
            OtpScreen(phone: state.uri.queryParameters['phone'] ?? ''),
      ),
      GoRoute(path: '/manager', builder: (_, _) => const ManagerShellScreen()),
      GoRoute(path: '/home', builder: (_, _) => const ResidentShellScreen()),
    ],
  );

  ref.listen(authControllerProvider, (_, _) => router.refresh());
  ref.onDispose(router.dispose);
  return router;
});

String? _redirect(Ref ref, GoRouterState state) {
  final auth = ref.read(authControllerProvider);
  final location = state.matchedLocation;

  // Startup restore in flight — hold on the splash.
  if (auth.status == AuthStatus.unknown) {
    return location == '/' ? null : '/';
  }

  if (auth.status == AuthStatus.unauthenticated) {
    return location.startsWith('/login') ? null : '/login';
  }

  // Authenticated: never show splash/login; land each role on its own shell
  // and keep roles inside their own section.
  final isManager = auth.user?.role == UserRole.manager;
  final home = isManager ? '/manager' : '/home';
  final allowed = isManager ? {'/manager'} : {'/home'};
  return location == home || allowed.contains(location) ? null : home;
}
