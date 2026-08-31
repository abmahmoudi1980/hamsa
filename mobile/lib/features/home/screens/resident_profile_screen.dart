import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../auth/auth_controller.dart';
import '../../auth/models/user_session.dart';

/// US9 (T084) — Resident profile screen: role + name (read-only in P0),
/// logout. Name editing arrives once `PATCH /auth/me` lands (post-P0
/// follow-up); for now the screen surfaces the user object returned by
/// `POST /auth/verify` so the resident can confirm their identity and sign
/// out.
class ResidentProfileScreen extends ConsumerWidget {
  const ResidentProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final user = ref.watch(authControllerProvider).user;
    if (user == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    final role = user.role == UserRole.manager
        ? l10n.roleManager
        : l10n.roleResident;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.profileTitle)),
      body: ListView(
        padding: AppTheme.pagePadding,
        children: [
          ListTile(
            leading: const Icon(Icons.badge_outlined),
            title: Text(l10n.profileName),
            subtitle: Text(user.name.isEmpty ? '—' : user.name),
          ),
          ListTile(
            leading: const Icon(Icons.verified_user_outlined),
            title: Text('${l10n.roleManager} / ${l10n.roleResident}'),
            subtitle: Text(role),
          ),
          if (user.primaryBuildingId != null)
            ListTile(
              leading: const Icon(Icons.apartment),
              title: Text(l10n.buildingsTitle),
              subtitle: Text(user.primaryBuildingId!),
            ),
          const SizedBox(height: AppTheme.spaceXxl),
          OutlinedButton.icon(
            onPressed: () =>
                ref.read(authControllerProvider.notifier).logout(),
            icon: const Icon(Icons.logout),
            label: Text(l10n.logout),
          ),
        ],
      ),
    );
  }
}

// Suppress unused warning for helper typedef.
// ignore: unused_element
typedef _UserRef = UserSession;
