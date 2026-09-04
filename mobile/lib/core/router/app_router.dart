import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/auth_controller.dart';
import '../../features/auth/models/user_session.dart';
import '../../features/auth/screens/login_screen.dart';
import '../../features/auth/screens/manager_invite_screen.dart';
import '../../features/auth/screens/register_screen.dart';
import '../../features/auth/screens/setup_screen.dart';
import '../../features/buildings/screens/building_form_screen.dart';
import '../../features/buildings/screens/building_list_screen.dart';
import '../../features/buildings/screens/building_managers_screen.dart';
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
import '../../features/expenses/screens/expense_form_screen.dart';
import '../../features/expenses/screens/expense_list_screen.dart';
import '../../features/expenses/screens/financial_report_screen.dart';
import '../../features/announcements/models/announcement.dart';
import '../../features/announcements/screens/announcement_form_screen.dart';
import '../../features/announcements/screens/manager_announcement_list_screen.dart';
import '../../features/dashboard/screens/manager_dashboard_screen.dart';
import '../../features/announcements/screens/resident_announcement_detail_screen.dart';
import '../../features/announcements/screens/resident_announcement_list_screen.dart';
import '../../features/home/screens/resident_shell_screen.dart';
import '../../features/maintenance/screens/manager_maintenance_detail_screen.dart';
import '../../features/maintenance/screens/manager_maintenance_list_screen.dart';
import '../../features/maintenance/screens/resident_maintenance_form_screen.dart';
import '../../features/maintenance/screens/resident_maintenance_list_screen.dart';
import '../../features/payments/screens/payment_history_screen.dart';
import '../../features/payments/screens/payment_ledger_screen.dart';
import '../../features/payments/screens/record_payment_screen.dart';
import '../../features/payments/screens/select_invoice_screen.dart';
import '../../features/notifications/screens/notification_center_screen.dart';

import '../../features/payments/screens/pay_invoice_screen.dart';

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
      GoRoute(path: '/register', builder: (_, _) => const RegisterScreen()),
      GoRoute(path: '/setup', builder: (_, _) => const SetupScreen()),
      // Manager section (US2 buildings & units; US3 people & occupancy;
      // dashboard arrives in US10).
      GoRoute(
        path: '/manager',
        builder: (_, _) => const ManagerDashboardScreen(),
        routes: [
          GoRoute(
            path: 'buildings',
            builder: (_, _) => const BuildingListScreen(),
            routes: [
              GoRoute(
                path: 'new',
                builder: (_, _) => const BuildingFormScreen(),
              ),
            ],
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
            path: 'buildings/:buildingId/managers',
            builder: (_, state) => BuildingManagersScreen(
              buildingId: state.pathParameters['buildingId']!,
            ),
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
            path: 'invite',
            builder: (_, _) => const ManagerInviteScreen(),
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
        path: '/manager/select-invoice/:buildingId',
        builder: (_, state) => SelectInvoiceScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
      ),
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
      // US6 (T066/T067): expenses and the financial report — manager only.
      GoRoute(
        path: '/manager/expenses/:buildingId',
        builder: (_, state) => ExpenseListScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
        routes: [
          GoRoute(
            path: 'new',
            builder: (_, state) => ExpenseFormScreen(
              buildingId: state.pathParameters['buildingId']!,
            ),
          ),
          GoRoute(
            path: ':expenseId',
            builder: (_, state) => ExpenseFormScreen(
              buildingId: state.pathParameters['buildingId']!,
              expenseId: state.pathParameters['expenseId'],
            ),
          ),
        ],
      ),
      GoRoute(
        path: '/manager/report/:buildingId',
        builder: (_, state) => FinancialReportScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
      ),
      // US7 (T073/T074): maintenance — resident submit/track + manager workflow.
      GoRoute(
        path: '/home/maintenance',
        builder: (_, _) => const ResidentMaintenanceListScreen(),
      ),
      GoRoute(
        path: '/home/maintenance/new',
        builder: (_, _) => const ResidentMaintenanceFormScreen(),
      ),
      GoRoute(
        path: '/manager/maintenance/:buildingId',
        builder: (_, state) => ManagerMaintenanceListScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
      ),
      GoRoute(
        path: '/manager/maintenance/:buildingId/:requestId',
        builder: (_, state) => ManagerMaintenanceDetailScreen(
          buildingId: state.pathParameters['buildingId']!,
          requestId: state.pathParameters['requestId']!,
        ),
      ),
      // US8 (T079/T080): announcements — manager publish + resident targeted list + notification center.
      GoRoute(
        path: '/manager/announcements/:buildingId',
        builder: (_, state) => ManagerAnnouncementListScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
      ),
      GoRoute(
        path: '/manager/announcements/:buildingId/new',
        builder: (_, state) => AnnouncementFormScreen(
          buildingId: state.pathParameters['buildingId']!,
        ),
      ),
      GoRoute(
        path: '/manager/announcements/:buildingId/:announcementId',
        builder: (_, state) => AnnouncementFormScreen(
          buildingId: state.pathParameters['buildingId']!,
          announcement: state.extra is Announcement ? state.extra as Announcement : null,
        ),
      ),
      GoRoute(
        path: '/home/announcements',
        builder: (_, _) => const ResidentAnnouncementListScreen(),
      ),
      GoRoute(
        path: '/home/announcements/:id',
        builder: (_, state) => ResidentAnnouncementDetailScreen(
          announcementId: state.pathParameters['id']!,
          announcement: state.extra is Announcement ? state.extra as Announcement : null,
        ),
      ),
      GoRoute(
        path: '/home/notifications',
        builder: (_, _) => const NotificationCenterScreen(),
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
    const authPaths = ['/login', '/register', '/setup'];
    final isAuthPath =
        authPaths.any((p) => location == p || location.startsWith('$p/'));
    return isAuthPath ? null : '/login';
  }
  // Authenticated: never show splash/login — post-login and app-start land
  // each role on its own shell; keep roles out of the other role's section.
  // Shared surfaces (/invoice/:id, /invoice/:id/pay) are reachable from both
  // sections.
  // 002-multi-manager-support: the superadmin governs no buildings but
  // shares the manager shell for the invite tool (its home and the building
  // lists are guarded client-side — `GET /buildings` is manager-only).
  final role = auth.user?.role;
  final isManager = role == UserRole.manager || role == UserRole.superadmin;
  final home = isManager ? '/manager' : '/home';
  final otherSection = isManager ? '/home' : '/manager';
  if (location == '/' ||
      location.startsWith('/login') ||
      location.startsWith('/register') ||
      location.startsWith('/setup')) {
    return home;
  }
  return location.startsWith(otherSection) ? home : null;
}
