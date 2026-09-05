import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/empty_state.dart';
import '../buildings_controller.dart';
import '../models/building.dart';

/// Change history of a unit from the append-only audit trail (FR-004).
class UnitHistoryScreen extends ConsumerWidget {
  const UnitHistoryScreen({super.key, required this.unitId});

  final String unitId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final repo = ref.watch(buildingsRepositoryProvider);
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.changeHistory)),
      body: SafeArea(
        child: FutureBuilder<List<UnitHistoryEntry>>(
          future: repo.unitHistory(unitId),
          builder: (context, snap) {
            if (snap.connectionState != ConnectionState.done) {
              return const Center(child: CircularProgressIndicator());
            }
            final items = snap.data ?? const [];
            if (items.isEmpty) {
              return EmptyState(title: l10n.noChangeHistory);
            }
            return ListView.separated(
              padding: AppTheme.pagePadding,
              itemCount: items.length,
              separatorBuilder: (_, _) => const SizedBox(height: 10),
              itemBuilder: (context, i) => Card(
                child: ListTile(
                  leading: Container(
                    width: 44,
                    height: 44,
                    decoration: BoxDecoration(
                      color: scheme.primary.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(AppTheme.radiusSmall),
                    ),
                    child: const Icon(Icons.history, size: 22),
                  ),
                  title: Text(items[i].action),
                ),
              ),
            );
          },
        )
      ),
    );
  }
}
