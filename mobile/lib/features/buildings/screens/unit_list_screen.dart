import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/empty_state.dart';
import '../buildings_controller.dart';

/// Searchable/filterable unit list for one building (US2/T034).
/// Filters: free text (?q) and status (?status=); page envelope total shown
/// in the app bar body header.
class UnitListScreen extends ConsumerStatefulWidget {
  const UnitListScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  ConsumerState<UnitListScreen> createState() => _UnitListScreenState();
}

class _UnitListScreenState extends ConsumerState<UnitListScreen> {
  final _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  UnitsFilter _filter(WidgetRef ref) =>
      ref
          .read(unitsControllerProvider(widget.buildingId))
          .valueOrNull
          ?.filter ??
      const UnitsFilter();

  void _apply(WidgetRef ref, UnitsFilter f) {
    ref.read(unitsControllerProvider(widget.buildingId).notifier).apply(f);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(unitsControllerProvider(widget.buildingId));
    final status = async.valueOrNull?.filter.status ?? '';
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.unitsTitle),
        actions: [
          // US5 (T060): the building's payment ledger.
          IconButton(
            icon: const Icon(Icons.payments_outlined),
            tooltip: l10n.ledgerTitle,
            onPressed: () =>
                context.push('/manager/ledger/${widget.buildingId}'),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          await context.push(
            '/manager/buildings/${widget.buildingId}/units/new',
          );
          if (mounted) {
            ref.invalidate(unitsControllerProvider(widget.buildingId));
          }
        },
        icon: const Icon(Icons.add),
        label: Text(l10n.addUnit),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(
              AppTheme.spaceM,
              AppTheme.spaceS,
              AppTheme.spaceM,
              0,
            ),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _search,
                    decoration: InputDecoration(
                      hintText: l10n.searchUnits,
                      prefixIcon: const Icon(Icons.search),
                    ),
                    onSubmitted: (_) =>
                        _apply(ref, _filter(ref).copyWith(q: _search.text)),
                  ),
                ),
                const SizedBox(width: AppTheme.spaceS),
                DropdownButton<String>(
                  value: status.isEmpty ? null : status,
                  hint: Text(l10n.allStatuses),
                  items: [
                    DropdownMenuItem(
                      value: 'active',
                      child: Text(l10n.statusActive),
                    ),
                    DropdownMenuItem(
                      value: 'vacant',
                      child: Text(l10n.statusVacant),
                    ),
                    DropdownMenuItem(
                      value: 'occupied',
                      child: Text(l10n.statusOccupied),
                    ),
                    DropdownMenuItem(
                      value: 'inactive',
                      child: Text(l10n.statusInactive),
                    ),
                  ],
                  onChanged: (v) => _apply(
                    ref,
                    UnitsFilter(status: v ?? '', q: _search.text),
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: async.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, _) =>
                  EmptyState(icon: Icons.error_outline, title: l10n.errorUnknown),
              data: (s) => s.units.isEmpty
                  ? EmptyState(
                      title: l10n.emptyStateTitle,
                      actionLabel: l10n.addUnit,
                      onAction: () async {
                        await context.push(
                          '/manager/buildings/${widget.buildingId}/units/new',
                        );
                        if (mounted) {
                          ref.invalidate(
                            unitsControllerProvider(widget.buildingId),
                          );
                        }
                      },
                    )
                  : ListView.separated(
                      padding: AppTheme.pagePadding,
                      itemCount: s.units.length,
                      separatorBuilder: (_, _) => const SizedBox(height: 10),
                      itemBuilder: (context, i) {
                        final u = s.units[i];
                        return Card(
                          child: ListTile(
                            leading: Container(
                              width: 44,
                              height: 44,
                              alignment: Alignment.center,
                              decoration: BoxDecoration(
                                color: scheme.primary.withValues(alpha: 0.1),
                                borderRadius: BorderRadius.circular(
                                  AppTheme.radiusSmall,
                                ),
                              ),
                              child: Text(
                                u.number,
                                style: Theme.of(context).textTheme.titleSmall,
                              ),
                            ),
                            title: Text(
                              '${l10n.unitFloor} ${u.floor}'
                              '${(u.block == null || u.block!.isEmpty) ? '' : ' • ${u.block}'}',
                            ),
                            subtitle: Text('${u.areaM2} ${l10n.areaM2}'),
                            trailing: Chip(
                              label: Text(_statusLabel(l10n, u.status)),
                            ),
                            onTap: () async {
                              await context.push(
                                '/manager/buildings/${widget.buildingId}/units/${u.id}',
                              );
                              ref.invalidate(
                                unitsControllerProvider(widget.buildingId),
                              );
                            },
                          ),
                        );
                      },
                    ),
            ),
          ),
        ],
      ),
    );
  }
}

String _statusLabel(AppLocalizations l10n, String s) => switch (s) {
  'vacant' => l10n.statusVacant,
  'occupied' => l10n.statusOccupied,
  'inactive' => l10n.statusInactive,
  _ => l10n.statusActive,
};
