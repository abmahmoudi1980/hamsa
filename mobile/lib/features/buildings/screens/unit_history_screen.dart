import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/l10n/app_localizations.dart';
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

    return Scaffold(
      appBar: AppBar(title: Text(l10n.changeHistory)),
      body: FutureBuilder<List<UnitHistoryEntry>>(
        future: repo.unitHistory(unitId),
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          final items = snap.data ?? const [];
          if (items.isEmpty) {
            return Center(child: Text(l10n.noChangeHistory));
          }
          return ListView.separated(
            itemCount: items.length,
            separatorBuilder: (_, _) => const Divider(height: 1),
            itemBuilder: (context, i) => ListTile(
              leading: const Icon(Icons.history),
              title: Text(items[i].action),
            ),
          );
        },
      ),
    );
  }
}
