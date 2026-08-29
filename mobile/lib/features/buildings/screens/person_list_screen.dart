import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../people_controller.dart';

/// Person list for one building (US3/T040). Tiles open the edit form; the
/// FAB opens the create form. Persian copy from fa.arb via l10n.
class PersonListScreen extends ConsumerWidget {
  const PersonListScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final async = ref.watch(peopleControllerProvider(buildingId));

    return Scaffold(
      appBar: AppBar(title: Text(l10n.peopleTitle)),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          await context.push('/manager/buildings/$buildingId/people/new');
          if (context.mounted) {
            ref.invalidate(peopleControllerProvider(buildingId));
          }
        },
        icon: const Icon(Icons.add),
        label: Text(l10n.addPerson),
      ),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(child: Text(l10n.errorUnknown)),
        data: (people) => people.isEmpty
            ? Center(child: Text(l10n.emptyStateTitle))
            : ListView.separated(
                itemCount: people.length,
                separatorBuilder: (_, _) => const Divider(height: 1),
                itemBuilder: (context, i) {
                  final p = people[i];
                  return ListTile(
                    leading: CircleAvatar(
                      child: Text(
                        p.fullName.isEmpty ? '?' : p.fullName.substring(0, 1),
                      ),
                    ),
                    title: Text(p.fullName),
                    subtitle: Text(
                      [
                        if (p.phone != null && p.phone!.isNotEmpty) p.phone!,
                        if (p.nationalId != null && p.nationalId!.isNotEmpty)
                          p.nationalId!,
                      ].join(' • '),
                    ),
                    onTap: () async {
                      await context.push(
                        '/manager/buildings/$buildingId/people/${p.id}',
                      );
                      ref.invalidate(peopleControllerProvider(buildingId));
                    },
                  );
                },
              ),
      ),
    );
  }
}
