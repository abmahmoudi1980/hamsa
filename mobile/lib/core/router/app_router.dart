import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/auth_controller.dart';
import '../../features/auth/models/user_session.dart';
import '../../features/auth/screens/login_screen.dart';
import '../../features/auth/screens/otp_screen.dart';
import '../../features/buildings/screens/building_form_screen.dart';
import '../../features/buildings/screens/building_list_screen.dart';
import '../../features/buildings/screens/occupancy_form_screen.dart';
import '../../features/buildings/screens/person_form_screen.dart';
import '../../features/buildings/screens/person_list_screen.dart';
import '../../features/buildings/screens/unit_form_screen.dart';
import '../../features/buildings/screens/unit_history_screen.dart';
import '../../features/buildings/screens/unit_list_screen.dart';
import '../../features/buildings/screens/unit_occupancy_screen.dart';
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
      // Manager section (US2 buildings & units; US3 people & occupancy;
      // dashboard arrives in US10).
      GoRoute(
        path: '/manager',
        builder: (_, _) => const BuildingListScreen(),
        routes: [
          GoRoute(
            path: 'buildings/new',
            builder: (_, _) => const BuildingFormScreen(),
          ),
          GoRoute(
            path: 'buildings/:buildingId/people',
            builder: (_, state) => PersonListScreen(
              buildingId: state.pathParameters['buildingId']!,
            ),
            routes: [
              GoRoute(
                path: 'new',
                builder: (_, state) => PersonFormScreen(
                  buildingId: state.pathParameters['buildingId']!,
                ),
              ),
              GoRoute(
                path: ':personId',
                builder: (_, state) => PersonFormScreen(
                  buildingId: state.pathParameters['buildingId']!,
                  personId: state.pathParameters['personId'],
                ),
              ),
            ],
          ),
          GoRoute(
            path: 'buildings/:buildingId/units',
            builder: (_, state) =>
                UnitListScreen(buildingId: state.pathParameters['buildingId']!),
            routes: [
              GoRoute(
                path: 'new',
                builder: (_, state) => UnitFormScreen(
                  buildingId: state.pathParameters['buildingId']!,
                ),
              ),
              GoRoute(
                path: ':unitId',
                builder: (_, state) => UnitFormScreen(
                  buildingId: state.pathParameters['buildingId']!,
                  unitId: state.pathParameters['unitId'],
                ),
                routes: [
                  GoRoute(
                    path: 'history',
                    builder: (_, state) => UnitHistoryScreen(
                      unitId: state.pathParameters['unitId']!,
                    ),
                  ),
                  GoRoute(
                    path: 'occupancy',
                    builder: (_, state) => UnitOccupancyScreen(
                      buildingId: state.pathParameters['buildingId']!,
                      unitId: state.pathParameters['unitId']!,
                    ),
                    routes: [
                      GoRoute(
                        path: 'new',
                        builder: (_, state) => OccupancyFormScreen(
                          buildingId: state.pathParameters['buildingId']!,
                          unitId: state.pathParameters['unitId']!,
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
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
  final section = isManager ? '/manager' : '/home';
  return location.startsWith(section) ? null : home;
}
