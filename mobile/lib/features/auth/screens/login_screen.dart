import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';
import 'package:hamsa/core/theme/app_theme.dart';
import 'package:hamsa/shared/validation/validators.dart';

import '../auth_controller.dart';
import '../auth_repository.dart';

/// US1 — phone + password login. Residents without an account reach
/// [RegisterScreen] via the invite-code link; a fresh deployment's manager
/// starts at [SetupScreen].
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _obscure = true;
  bool _submitting = false;

  @override
  void dispose() {
    _phoneController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);
    final phone = fromPersianDigits(_phoneController.text.trim());

    try {
      final result = await ref
          .read(authRepositoryProvider)
          .login(phone: phone, password: _passwordController.text);
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
    final theme = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            child: Padding(
              padding: AppTheme.pagePadding,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Brand mark: tonal primary square with the building glyph.
                  Center(
                    child: Container(
                      width: 72,
                      height: 72,
                      decoration: BoxDecoration(
                        color: theme.colorScheme.primary,
                        borderRadius: BorderRadius.circular(
                          AppTheme.radiusLarge,
                        ),
                      ),
                      child: const Icon(
                        Icons.apartment,
                        size: 36,
                        color: Colors.white,
                      ),
                    ),
                  ),
                  const SizedBox(height: AppTheme.spaceXl),
                  Text(
                    l10n.loginTitle,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.displaySmall,
                  ),
                  const SizedBox(height: AppTheme.spaceS),
                  Text(
                    l10n.loginSubtitle,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: AppTheme.spaceXxl),
                  Card(
                    margin: EdgeInsets.zero,
                    child: Padding(
                      padding: const EdgeInsets.all(AppTheme.spaceXl),
                      child: Form(
                        key: _formKey,
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            TextFormField(
                              controller: _phoneController,
                              decoration: InputDecoration(
                                labelText: l10n.phoneLabel,
                              ),
                              keyboardType: TextInputType.phone,
                              autofocus: true,
                              validator: (v) => Validators.mobile(l10n, v),
                              onFieldSubmitted: (_) => _submit(),
                            ),
                            const SizedBox(height: AppTheme.spaceL),
                            TextFormField(
                              controller: _passwordController,
                              decoration: InputDecoration(
                                labelText: l10n.passwordLabel,
                                suffixIcon: IconButton(
                                  icon: Icon(
                                    _obscure
                                        ? Icons.visibility_outlined
                                        : Icons.visibility_off_outlined,
                                  ),
                                  onPressed: () =>
                                      setState(() => _obscure = !_obscure),
                                ),
                              ),
                              obscureText: _obscure,
                              validator: (v) => v == null || v.isEmpty
                                  ? l10n.requiredField
                                  : null,
                              onFieldSubmitted: (_) => _submit(),
                            ),
                            const SizedBox(height: AppTheme.spaceL),
                            SizedBox(
                              width: double.infinity,
                              child: FilledButton(
                                onPressed: _submitting ? null : _submit,
                                child: _submitting
                                    ? const SizedBox(
                                        width: 20,
                                        height: 20,
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2,
                                        ),
                                      )
                                    : Text(l10n.loginButton),
                              ),
                            ),
                            const SizedBox(height: AppTheme.spaceS),
                            TextButton(
                              onPressed: () => context.push('/register'),
                              child: Text(l10n.gotoRegister),
                            ),
                            TextButton(
                              onPressed: () => context.push('/setup'),
                              child: Text(
                                l10n.gotoSetup,
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
