import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';
import 'package:hamsa/core/theme/app_theme.dart';
import 'package:hamsa/shared/validation/validators.dart';

import '../auth_controller.dart';
import '../auth_repository.dart';

/// First-account bootstrap: on a fresh deployment the first account created
/// here becomes the building manager. Once a manager exists the backend
/// answers 409 and the screen shows the error.
class SetupScreen extends ConsumerStatefulWidget {
  const SetupScreen({super.key});

  @override
  ConsumerState<SetupScreen> createState() => _SetupScreenState();
}

class _SetupScreenState extends ConsumerState<SetupScreen> {
  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  final _nameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _confirmController = TextEditingController();
  bool _obscure = true;
  bool _submitting = false;

  @override
  void dispose() {
    _phoneController.dispose();
    _nameController.dispose();
    _passwordController.dispose();
    _confirmController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);

    try {
      final result = await ref.read(authRepositoryProvider).setup(
            phone: fromPersianDigits(_phoneController.text.trim()),
            password: _passwordController.text,
            name: _nameController.text.trim(),
          );
      if (!mounted) return;
      await ref
          .read(authControllerProvider.notifier)
          .completeLogin(
            user: result.user,
            accessToken: result.accessToken,
            refreshToken: result.refreshToken,
          );
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
    return Scaffold(
      appBar: AppBar(title: Text(l10n.setupTitle)),
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
                      Text(l10n.setupSubtitle),
                      const SizedBox(height: AppTheme.spaceL),
                      TextFormField(
                        controller: _phoneController,
                        decoration: InputDecoration(
                          labelText: l10n.phoneLabel,
                        ),
                        keyboardType: TextInputType.phone,
                        autofocus: true,
                        validator: (v) => Validators.mobile(l10n, v),
                      ),
                      const SizedBox(height: AppTheme.spaceL),
                      TextFormField(
                        controller: _nameController,
                        decoration: InputDecoration(
                          labelText: l10n.nameLabel,
                        ),
                        textInputAction: TextInputAction.next,
                      ),
                      const SizedBox(height: AppTheme.spaceL),
                      TextFormField(
                        controller: _passwordController,
                        decoration: InputDecoration(
                          labelText: l10n.passwordLabel,
                          suffixIcon: IconButton(
                            icon: Icon(_obscure
                                ? Icons.visibility_outlined
                                : Icons.visibility_off_outlined),
                            onPressed: () =>
                                setState(() => _obscure = !_obscure),
                          ),
                        ),
                        obscureText: _obscure,
                        validator: (v) => Validators.password(l10n, v),
                      ),
                      const SizedBox(height: AppTheme.spaceL),
                      TextFormField(
                        controller: _confirmController,
                        decoration: InputDecoration(
                          labelText: l10n.confirmPasswordLabel,
                        ),
                        obscureText: _obscure,
                        validator: (v) =>
                            v == _passwordController.text
                                ? null
                                : l10n.passwordsMismatch,
                      ),
                      const SizedBox(height: AppTheme.spaceXl),
                      FilledButton(
                        onPressed: _submitting ? null : _submit,
                        child: _submitting
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child:
                                    CircularProgressIndicator(strokeWidth: 2),
                              )
                            : Text(l10n.setupButton),
                      ),
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
