import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../../shared/widgets/menu_card.dart';
import 'resident_home_screen.dart';
import 'resident_profile_screen.dart';

/// US9 (T084) — Resident navigation shell with bottom nav for the full
/// 6-item menu: خانه، شارژها، پرداخت‌ها، تعمیرات، اطلاعیه‌ها، پروفایل.
class ResidentShellScreen extends ConsumerStatefulWidget {
  const ResidentShellScreen({super.key});

  @override
  ConsumerState<ResidentShellScreen> createState() =>
      _ResidentShellScreenState();
}

class _ResidentShellScreenState extends ConsumerState<ResidentShellScreen> {
  int _index = 0;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);

    final tabs = <_Tab>[
      _Tab(
        label: l10n.navHome,
        icon: Icons.home_outlined,
        selectedIcon: Icons.home,
        body: const ResidentHomeScreen(),
      ),
      _Tab(
        label: l10n.navCharges,
        icon: Icons.receipt_long_outlined,
        selectedIcon: Icons.receipt_long,
        // The charges/payments/maintenance/announcements tabs are stand-alone
        // routes handled by GoRouter; tapping the tab pushes the user into
        // the dedicated screen (which has its own back-stack). The shell
        // remains the home anchor for back-navigation.
        body: _RedirectShortcut(
          icon: Icons.receipt_long,
          title: l10n.myCharges,
          subtitle: l10n.homeViewAll,
          onTap: () => context.go('/home/charges'),
        ),
      ),
      _Tab(
        label: l10n.navPayments,
        icon: Icons.payments_outlined,
        selectedIcon: Icons.payments,
        body: _RedirectShortcut(
          icon: Icons.payments,
          title: l10n.paymentHistoryTitle,
          subtitle: l10n.homeViewAll,
          onTap: () => context.go('/home/payments'),
        ),
      ),
      _Tab(
        label: l10n.navMaintenance,
        icon: Icons.build_outlined,
        selectedIcon: Icons.build,
        body: _RedirectShortcut(
          icon: Icons.build,
          title: l10n.maintenanceResidentMenu,
          subtitle: l10n.homeViewAll,
          onTap: () => context.go('/home/maintenance'),
        ),
      ),
      _Tab(
        label: l10n.navAnnouncements,
        icon: Icons.campaign_outlined,
        selectedIcon: Icons.campaign,
        body: _RedirectShortcut(
          icon: Icons.campaign,
          title: l10n.navAnnouncements,
          subtitle: l10n.homeViewAll,
          onTap: () => context.go('/home/announcements'),
        ),
      ),
      _Tab(
        label: l10n.navProfile,
        icon: Icons.person_outline,
        selectedIcon: Icons.person,
        body: const ResidentProfileScreen(),
      ),
    ];

    return PopScope(
      canPop: _index == 0,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) return;
        setState(() => _index = 0);
      },
      child: Scaffold(
        body: IndexedStack(
          index: _index,
          children: [for (final t in tabs) t.body],
        ),
        bottomNavigationBar: NavigationBar(
          selectedIndex: _index,
          onDestinationSelected: (i) => setState(() => _index = i),
          destinations: [
            for (final t in tabs)
              NavigationDestination(
                icon: Icon(t.icon),
                selectedIcon: Icon(t.selectedIcon),
                label: t.label,
              ),
          ],
        ),
      ),
    );
  }
}

class _Tab {
  const _Tab({
    required this.label,
    required this.icon,
    required this.selectedIcon,
    required this.body,
  });
  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final Widget body;
}

class _RedirectShortcut extends StatelessWidget {
  const _RedirectShortcut({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          MenuCard(
            icon: icon,
            title: title,
            subtitle: subtitle,
            onTap: onTap,
          ),
        ],
      ),
    );
  }
}

// keep imports alive for future shortcuts
// ignore: unused_element
typedef _EmptyStateRef = EmptyState;
