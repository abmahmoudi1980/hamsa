import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/app_localizations.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets/empty_state.dart';
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
    final scheme = Theme.of(context).colorScheme;

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
        error: (e, _) =>
            EmptyState(icon: Icons.error_outline, title: l10n.errorUnknown),
        data: (people) => people.isEmpty
            ? EmptyState(
                title: l10n.emptyStateTitle,
                actionLabel: l10n.addPerson,
                onAction: () async {
                  await context.push(
                    '/manager/buildings/$buildingId/people/new',
                  );
                  if (context.mounted) {
                    ref.invalidate(peopleControllerProvider(buildingId));
                  }
                },
              )
            : ListView.separated(
                padding: AppTheme.pagePadding,
                itemCount: people.length,
                separatorBuilder: (_, _) => const SizedBox(height: 10),
                itemBuilder: (context, i) {
                  final p = people[i];
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
                          p.fullName.isEmpty ? '?' : p.fullName.substring(0, 1),
                          style: Theme.of(context).textTheme.titleSmall,
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
                    ),
                  );
                },
              ),
      ),
    );
  }
}
