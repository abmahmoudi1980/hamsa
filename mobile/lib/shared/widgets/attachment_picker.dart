import 'dart:io';

import 'package:flutter/material.dart';
import 'package:hamsa/core/l10n/app_localizations.dart';
import 'package:hamsa/core/storage/token_storage.dart';
import 'package:image_picker/image_picker.dart';

/// Picks a single image attachment (receipts, request photos) via the system
/// gallery picker and shows a removable preview.
class AttachmentPicker extends StatefulWidget {
  const AttachmentPicker({
    super.key,
    this.initial,
    this.existingUrl,
    required this.onChanged,
  });

  /// Already-attached file, if any.
  final XFile? initial;

  /// Server-stored attachment (GET /files/{id} URL), shown until a new file
  /// is picked. Fetched with the session's Authorization header.
  final String? existingUrl;

  /// Called whenever the attachment is added or removed (null = removed).
  final ValueChanged<XFile?> onChanged;

  @override
  State<AttachmentPicker> createState() => _AttachmentPickerState();
}

class _AttachmentPickerState extends State<AttachmentPicker> {
  XFile? _file;
  final ImagePicker _picker = ImagePicker();
  Map<String, String> _headers = const {};

  @override
  void initState() {
    super.initState();
    if (widget.existingUrl != null) _loadHeaders();
  }

  @override
  void didUpdateWidget(covariant AttachmentPicker oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.existingUrl != oldWidget.existingUrl) _loadHeaders();
  }

  Future<void> _loadHeaders() async {
    final session = await TokenStorage().read();
    if (!mounted) return;
    setState(() => _headers = {
          if (session != null)
            'Authorization': 'Bearer ${session.accessToken}',
        });
  }

  Future<void> _pick() async {
    final picked = await _picker.pickImage(source: ImageSource.gallery, imageQuality: 85);
    if (picked == null) return;
    setState(() => _file = picked);
    widget.onChanged(picked);
  }

  void _remove() {
    setState(() => _file = null);
    widget.onChanged(null);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final file = _file ?? widget.initial;

    if (file == null) {
      final url = widget.existingUrl;
      if (url != null) {
        return _preview(
          Image.network(
            url,
            headers: _headers,
            fit: BoxFit.cover,
            errorBuilder: (_, __, ___) => const SizedBox(
                height: 80, child: Icon(Icons.broken_image_outlined)),
          ),
        );
      }
      return OutlinedButton.icon(
        onPressed: _pick,
        icon: const Icon(Icons.image_outlined),
        label: Text(l10n.attachImage),
      );
    }

    return _preview(
      Image.file(
        File(file.path),
        fit: BoxFit.cover,
        errorBuilder: (_, __, ___) =>
            const SizedBox(height: 80, child: Icon(Icons.broken_image_outlined)),
      ),
    );
  }

  Widget _preview(Image image) {
    final l10n = AppLocalizations.of(context);
    return Stack(
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 180),
            child: image,
          ),
        ),
        Positioned(
          top: 4,
          left: 4,
          child: Material(
            color: Theme.of(context).colorScheme.surface.withValues(alpha: 0.9),
            shape: const CircleBorder(),
            child: IconButton(
              tooltip: l10n.removeAttachment,
              icon: const Icon(Icons.close, size: 20),
              onPressed: _remove,
            ),
          ),
        ),
      ],
    );
  }
}

