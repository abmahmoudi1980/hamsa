import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/datetime/jalali.dart';
import '../../../core/l10n/app_localizations.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/validation/validators.dart';
import '../../../shared/widgets/confirm_dialog.dart';
import '../../../shared/widgets/empty_state.dart';
import '../buildings_controller.dart';
import '../models/building.dart';

/// Per-building managers (US5/T029, contracts/api.md 002 delta): list of
/// granted managers with Jalali grant dates, add-by-phone, and remove with
/// confirm. Server 409/400/404/403 Persian messages surface verbatim; the
/// [managerAlreadyGrantedError]/[lastManagerError]/[selfRemoveError] strings
/// are the fallback when an envelope carries no message.
class BuildingManagersScreen extends ConsumerStatefulWidget {
  const BuildingManagersScreen({super.key, required this.buildingId});

  final String buildingId;

  @override
  ConsumerState<BuildingManagersScreen> createState() =>
      _BuildingManagersScreenState();
}

class _BuildingManagersScreenState extends ConsumerState<BuildingManagersScreen> {
  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _phoneController.dispose();
    super.dispose();
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(message)));
  }

  /// Server copy wins verbatim (no client-side re-translation); the
  /// status-specific [conflict]/[badRequest] fallbacks and the generic
  /// [describeError] mapping cover message-less envelopes.
  String _errorText(AppLocalizations l10n, Object e,
      {String? conflict, String? badRequest}) {
    final server = e is ApiException
        ? e.serverMessage
        : (e is DioException ? ApiException.from(e).serverMessage : null);
    if (server != null && server.trim().isNotEmpty) return server;
    final status = e is ApiException
        ? e.statusCode
        : (e is DioException ? e.response?.statusCode : null);
    if (status == 409 && conflict != null) return conflict;
    if (status == 400 && badRequest != null) return badRequest;
    return describeError(l10n, e);
  }

  Future<void> _add() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);

    try {
      await ref
          .read(managersControllerProvider(widget.buildingId).notifier)
          .add(fromPersianDigits(_phoneController.text.trim()));
      _phoneController.clear();
      messenger.showSnackBar(SnackBar(content: Text(l10n.managerAdded)));
    } catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            _errorText(l10n, e, conflict: l10n.managerAlreadyGrantedError),
          ),
          duration: const Duration(seconds: 8),
        ),
      );
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  Future<void> _remove(BuildingManager manager) async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await ConfirmDialog.show(
      context,
      title: l10n.removeManagerTitle,
      message: l10n.removeManagerBody,
      confirmLabel: l10n.delete,
      destructive: true,
    );
    if (confirmed != true || !mounted) return;

    try {
      await ref
          .read(managersControllerProvider(widget.buildingId).notifier)
          .remove(manager.userId);
      if (!mounted) return;
      _showMessage(l10n.managerRemoved);
    } catch (e) {
      if (!mounted) return;
      _showMessage(
        _errorText(
          l10n,
          e,
          conflict: l10n.lastManagerError,
          badRequest: l10n.selfRemoveError,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final scheme = Theme.of(context).colorScheme;
    final async = ref.watch(managersControllerProvider(widget.buildingId));

    return Scaffold(
      appBar: AppBar(title: Text(l10n.buildingManagersTitle)),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: AppTheme.pagePadding,
              child: Card(
                margin: EdgeInsets.zero,
                child: Padding(
                  padding: const EdgeInsets.all(AppTheme.spaceL),
                  child: Form(
                    key: _formKey,
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(
                          child: TextFormField(
                            controller: _phoneController,
                            decoration: InputDecoration(
                              labelText: l10n.phoneLabel,
                            ),
                            keyboardType: TextInputType.phone,
                            validator: (v) => Validators.mobile(l10n, v),
                            onFieldSubmitted: (_) => _add(),
                          ),
                        ),
                        const SizedBox(width: AppTheme.spaceS),
                        Padding(
                          padding: const EdgeInsets.only(top: AppTheme.spaceS),
                          child: FilledButton(
                            onPressed: _submitting ? null : _add,
                            child: _submitting
                                ? const SizedBox(
                                    width: 20,
                                    height: 20,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                    ),
                                  )
                                : Text(l10n.addManager),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
            Expanded(
              child: async.when(
                loading: () =>
                    const Center(child: CircularProgressIndicator()),
                error: (e, _) => EmptyState(
                  icon: Icons.error_outline,
                  title: e is ApiException
                      ? (e.serverMessage ?? l10n.apiErrorMessage(e))
                      : l10n.errorUnknown,
                  actionLabel: l10n.retry,
                  onAction: () => ref
                      .invalidate(managersControllerProvider(widget.buildingId)),
                ),
                data: (managers) => managers.isEmpty
                    ? EmptyState(
                        icon: Icons.engineering_outlined,
                        title: l10n.managersEmpty,
                      )
                    : RefreshIndicator(
                        onRefresh: () async => ref.invalidate(
                          managersControllerProvider(widget.buildingId),
                        ),
                        child: ListView.separated(
                          padding: AppTheme.pagePadding,
                          itemCount: managers.length,
                          separatorBuilder: (_, _) =>
                              const SizedBox(height: 10),
                          itemBuilder: (context, i) {
                            final m = managers[i];
                            final granted = DateTime.tryParse(m.grantedAt);
                            return Card(
                              child: ListTile(
                                leading: Container(
                                  width: 44,
                                  height: 44,
                                  alignment: Alignment.center,
                                  decoration: BoxDecoration(
                                    color:
                                        scheme.primary.withValues(alpha: 0.1),
                                    borderRadius: BorderRadius.circular(
                                      AppTheme.radiusSmall,
                                    ),
                                  ),
                                  child: Text(
                                    m.name.isEmpty
                                        ? '?'
                                        : m.name.substring(0, 1),
                                    style:
                                        Theme.of(context).textTheme.titleSmall,
                                  ),
                                ),
                                title: Text(
                                  m.name.isEmpty ? m.phone : m.name,
                                ),
                                subtitle: Text(
                                  [
                                    toPersianDigits(m.phone),
                                    if (granted != null)
                                      l10n.managersGrantedOn(
                                        formatJalaliDate(granted),
                                      ),
                                  ].join(' • '),
                                ),
                                trailing: IconButton(
                                  icon: const Icon(
                                    Icons.person_remove_outlined,
                                  ),
                                  tooltip: l10n.removeManagerTitle,
                                  onPressed: () => _remove(m),
                                ),
                              ),
                            );
                          },
                        ),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
