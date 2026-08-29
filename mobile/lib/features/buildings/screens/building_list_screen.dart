import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/empty_state.dart';
import '../../auth/auth_controller.dart';
import '../buildings_controller.dart';

/// Manager building list (US2/T033): every permitted building; tap → units;
/// FAB → create form.
class BuildingListScreen extends ConsumerWidget {
  const BuildingListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(buildingsControllerProvider);
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.buildingsTitle),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: l10n.logout,
            onPressed: () => ref.read(authControllerProvider.notifier).logout(),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/manager/buildings/new'),
        icon: const Icon(Icons.add),
        label: Text(l10n.addBuilding),
      ),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => EmptyState(
          icon: Icons.error_outline,
          title: e is ApiException
              ? (e.serverMessage ?? l10n.apiErrorMessage(e))
              : l10n.errorUnknown,
          actionLabel: l10n.retry,
          onAction: () => ref.invalidate(buildingsControllerProvider),
        ),
        data: (buildings) => buildings.isEmpty
            ? EmptyState(
                title: l10n.emptyStateTitle,
                subtitle: l10n.emptyStateSubtitle,
                actionLabel: l10n.addBuilding,
                onAction: () => context.push('/manager/buildings/new'),
              )
            : RefreshIndicator(
                onRefresh: () async =>
                    ref.invalidate(buildingsControllerProvider),
                child: ListView.separated(
                  padding: AppTheme.pagePadding,
                  itemCount: buildings.length,
                  separatorBuilder: (_, _) => const SizedBox(height: 10),
                  itemBuilder: (context, i) {
                    final b = buildings[i];
                    return Card(
                      child: ListTile(
                        leading: Container(
                          width: 44,
                          height: 44,
                          decoration: BoxDecoration(
                            color: scheme.primary.withValues(alpha: 0.1),
                            borderRadius: BorderRadius.circular(
                              AppTheme.radiusSmall,
                            ),
                          ),
                          child: const Icon(
                            Icons.apartment_outlined,
                            size: 22,
                          ),
                        ),
                        title: Text(b.name),
                        subtitle: Text(
                          '${b.unitCount} ${l10n.unitsTitle}'
                          '${b.address == null || b.address!.isEmpty ? '' : ' • ${b.address}'}',
                        ),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            // US4 (T050): billing periods entry point.
                            IconButton(
                              icon: const Icon(Icons.receipt_long),
                              tooltip: l10n.billingTitle,
                              onPressed: () =>
                                  context.push('/manager/periods/${b.id}'),
                            ),
                            const Icon(Icons.chevron_left),
                            // US6 (T066): expenses & financial report entry.
                            IconButton(
                              icon: const Icon(Icons.receipt),
                              tooltip: l10n.expensesTitle,
                              onPressed: () =>
                                  context.push('/manager/expenses/${b.id}'),
                            ),
                          ],
                        ),
                        onTap: () =>
                            context.push('/manager/buildings/${b.id}/units'),
                      ),
                    );
                  },
                ),
              ),
      ),
    );
  }
}
