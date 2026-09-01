import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../auth/models/user_session.dart';
import '../../buildings/buildings_controller.dart';
import '../../buildings/models/building.dart';
import '../../auth/auth_controller.dart';

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
            // Tile only renders when the session carries a building id
            // (managers); the name resolves from the manager-scoped
            // /buildings list, so the raw UUID never reaches the UI.
            // Loading/error fall back to a dash.
            ListTile(
              leading: const Icon(Icons.apartment),
              title: Text(l10n.buildingsTitle),
              subtitle: Text(
                ref
                        .watch(buildingsControllerProvider)
                        .maybeWhen(
                          data: (buildings) =>
                              _buildingName(buildings, user.primaryBuildingId!),
                          orElse: () => null,
                        ) ??
                    '—',
              ),
            ),
          const SizedBox(height: AppTheme.spaceXxl),
          OutlinedButton.icon(
            onPressed: () => ref.read(authControllerProvider.notifier).logout(),
            icon: const Icon(Icons.logout),
            label: Text(l10n.logout),
          ),
        ],
      ),
    );
  }
}

String? _buildingName(List<Building> buildings, String id) {
  for (final b in buildings) {
    if (b.id == id) return b.name;
  }
  return null;
}
