import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
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
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                e is ApiException
                    ? (e.serverMessage ?? l10n.apiErrorMessage(e))
                    : l10n.errorUnknown,
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () => ref.invalidate(buildingsControllerProvider),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        data: (buildings) => buildings.isEmpty
            ? EmptyState(onAdd: () => context.push('/manager/buildings/new'))
            : RefreshIndicator(
                onRefresh: () async =>
                    ref.invalidate(buildingsControllerProvider),
                child: ListView.separated(
                  itemCount: buildings.length,
                  separatorBuilder: (_, _) => const Divider(height: 1),
                  itemBuilder: (context, i) {
                    final b = buildings[i];
                    return ListTile(
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
                    );
                  },
                ),
              ),
      ),
    );
  }
}

class EmptyState extends StatelessWidget {
  const EmptyState({required this.onAdd, super.key});

  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            l10n.emptyStateTitle,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          FilledButton(onPressed: onAdd, child: Text(l10n.addBuilding)),
        ],
      ),
    );
  }
}
