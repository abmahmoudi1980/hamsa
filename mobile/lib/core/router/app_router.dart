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
import '../../features/billing/models/billing.dart';
import '../../features/billing/screens/cost_item_form_screen.dart';
import '../../features/billing/screens/period_detail_screen.dart';
import '../../features/billing/screens/period_form_screen.dart';
import '../../features/billing/screens/period_list_screen.dart';
import '../../features/charges/screens/charges_list_screen.dart';
import '../../features/charges/screens/invoice_detail_screen.dart';
import '../../features/payments/screens/payment_history_screen.dart';
import '../../features/payments/screens/payment_ledger_screen.dart';
import '../../features/payments/screens/pay_invoice_screen.dart';
import '../../features/payments/screens/record_payment_screen.dart';
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
          GoRoute(
            path: 'periods/:buildingId',
            builder: (_, state) => PeriodListScreen(
              buildingId: state.pathParameters['buildingId']!,
            ),
            routes: [
              GoRoute(
                path: 'new',
                builder: (_, state) => PeriodFormScreen(
                  buildingId: state.pathParameters['buildingId']!,
                ),
              ),
              GoRoute(
                path: ':periodId',
                builder: (_, state) => PeriodDetailScreen(
                  buildingId: state.pathParameters['buildingId']!,
                  periodId: state.pathParameters['periodId']!,
                ),
                routes: [
                  GoRoute(
                    path: 'cost-items/new',
                    builder: (_, state) => CostItemFormScreen(
                      buildingId: state.pathParameters['buildingId']!,
                      periodId: state.pathParameters['periodId']!,
                    ),
                  ),
                  GoRoute(
                    path: 'cost-items/:costItemId',
                    builder: (_, state) => CostItemFormScreen(
                      buildingId: state.pathParameters['buildingId']!,
                      periodId: state.pathParameters['periodId']!,
                      existing: state.extra is CostItem
                          ? state.extra as CostItem
                          : null,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
      // US5 payment surfaces. Manager: record manual payment on an invoice
      // and the building ledger. Resident: pay flow + payment history.
      GoRoute(
        path: '/manager/records-payment/:invoiceId',
        builder: (_, state) => RecordPaymentScreen(
          invoiceId: state.pathParameters['invoiceId']!,
        ),
      ),
      GoRoute(
        path: '/manager/ledger/:buildingId',
        builder: (_, state) => PaymentLedgerScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
      ),
      GoRoute(
        path: '/invoice/:invoiceId',
        builder: (_, state) => InvoiceDetailScreen(
          invoiceId: state.pathParameters['invoiceId']!,
        ),
      ),
      GoRoute(
        path: '/invoice/:invoiceId/pay',
        builder: (_, state) => PayInvoiceScreen(
          invoiceId: state.pathParameters['invoiceId']!,
        ),
      ),
      GoRoute(
        path: '/home/payments',
        builder: (_, _) => const PaymentHistoryScreen(),
      ),
      GoRoute(
        path: '/home/charges',
        builder: (_, _) => const ChargesListScreen(),
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
  // Authenticated: never show splash/login — post-login (OTP verified while
  // still on /login/otp) and app-start land each role on its own shell;
  // keep roles out of the other role's section. Shared surfaces
  // (/invoice/:id, /invoice/:id/pay) are reachable from both sections.
  final isManager = auth.user?.role == UserRole.manager;
  final home = isManager ? '/manager' : '/home';
  final otherSection = isManager ? '/home' : '/manager';
  if (location == '/' || location.startsWith('/login')) return home;
  return location.startsWith(otherSection) ? home : null;
}
