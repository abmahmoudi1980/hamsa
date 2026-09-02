import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';
import 'package:hamsa/core/theme/app_theme.dart';
import 'package:hamsa/shared/validation/validators.dart';

import '../auth_repository.dart';

/// Manager tool: issues a one-time invite code for a resident's phone. The
/// code is shown once with a copy button; the manager passes it to the
/// resident out-of-band.
class ManagerInviteScreen extends ConsumerStatefulWidget {
  const ManagerInviteScreen({super.key});

  @override
  ConsumerState<ManagerInviteScreen> createState() =>
      _ManagerInviteScreenState();
}

class _ManagerInviteScreenState extends ConsumerState<ManagerInviteScreen> {
  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  String? _code;
  bool _submitting = false;

  @override
  void dispose() {
    _phoneController.dispose();
    super.dispose();
  }

  Future<void> _issue() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);

    try {
      final code = await ref
          .read(authRepositoryProvider)
          .createInvite(fromPersianDigits(_phoneController.text.trim()));
      if (!mounted) return;
      setState(() => _code = code);
    } catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(describeError(l10n, e)),
          duration: const Duration(seconds: 8),
        ),
      );
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.inviteTitle)),
      body: SafeArea(
        child: SingleChildScrollView(
          child: Padding(
            padding: AppTheme.pagePadding,
            child: Card(
              margin: EdgeInsets.zero,
              child: Padding(
                padding: const EdgeInsets.all(AppTheme.spaceXl),
                child: Form(
                  key: _formKey,
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text(l10n.inviteDescription),
                      const SizedBox(height: AppTheme.spaceL),
                      TextFormField(
                        controller: _phoneController,
                        decoration: InputDecoration(
                          labelText: l10n.phoneLabel,
                        ),
                        keyboardType: TextInputType.phone,
                        autofocus: true,
                        validator: (v) => Validators.mobile(l10n, v),
                        onFieldSubmitted: (_) => _issue(),
                      ),
                      const SizedBox(height: AppTheme.spaceL),
                      FilledButton(
                        onPressed: _submitting ? null : _issue,
                        child: _submitting
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child:
                                    CircularProgressIndicator(strokeWidth: 2),
                              )
                            : Text(l10n.getInviteCode),
                      ),
                      if (_code != null) ...[
                        const SizedBox(height: AppTheme.spaceXl),
                        Container(
                          padding: const EdgeInsets.all(AppTheme.spaceL),
                          decoration: BoxDecoration(
                            color: theme.colorScheme.surfaceContainerHighest,
                            borderRadius:
                                BorderRadius.circular(AppTheme.radiusLarge),
                          ),
                          child: Row(
                            mainAxisAlignment:
                                MainAxisAlignment.spaceBetween,
                            children: [
                              Expanded(
                                child: Text(
                                  _code!,
                                  textAlign: TextAlign.center,
                                  style: theme.textTheme.headlineSmall
                                      ?.copyWith(
                                    letterSpacing: 4,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ),
                              IconButton(
                                tooltip: l10n.inviteCopied,
                                icon: const Icon(Icons.copy_outlined),
                                onPressed: () {
                                  Clipboard.setData(ClipboardData(text: _code!));
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    SnackBar(
                                        content: Text(l10n.inviteCopied)),
                                  );
                                },
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: AppTheme.spaceS),
                        Text(
                          l10n.inviteExpiresInDays(7),
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
