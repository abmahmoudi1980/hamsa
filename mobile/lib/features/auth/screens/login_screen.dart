import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:hamsa/core/datetime/jalali.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/network/api_exception.dart';
import 'package:hamsa/shared/validation/validators.dart';

import '../auth_repository.dart';

/// US1 step 1 — phone entry. Requests an OTP and hands the phone to the OTP
/// screen (T026). Persian digits typed by the user are normalized before the
/// API call.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _phoneController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _submitting = true);

    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context);
    final phone = fromPersianDigits(_phoneController.text.trim());

    try {
      await ref.read(authRepositoryProvider).requestOtp(phone);
      if (!mounted) return;
      context.push('/login/otp?phone=$phone');
    } on ApiException catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.apiErrorMessage(e))));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    l10n.loginTitle,
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    l10n.loginSubtitle,
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 24),
                  TextFormField(
                    controller: _phoneController,
                    decoration: InputDecoration(labelText: l10n.phoneLabel),
                    keyboardType: TextInputType.phone,
                    autofocus: true,
                    validator: (v) => Validators.mobile(l10n, v),
                    onFieldSubmitted: (_) => _submit(),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: _submitting ? null : _submit,
                    child: _submitting
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Text(l10n.requestOtp),
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
