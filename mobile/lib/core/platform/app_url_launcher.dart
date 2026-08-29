import 'package:flutter/services.dart';

/// US5: opens external URLs (payment-gateway pages) via the Android side.
/// `url_launcher` is unavailable offline; a two-line intent channel keeps
/// the same behavior without the dependency.
class AppUrlLauncher {
  static const _channel = MethodChannel('hamsa/launcher');

  /// Opens [url] in the external browser. Throws [PlatformException] when
  /// no activity can handle it (mapped to a Persian message by callers).
  static Future<void> launch(String url) async {
    await _channel.invokeMethod<void>('launchUrl', {'url': url});
  }
}
