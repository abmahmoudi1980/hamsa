package app.hamsa.hamsa

import android.content.Intent
import android.net.Uri
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

/// US5: opens the payment-gateway page in the device browser (Iranian
/// gateways are browser flows; no in-app webview dependency). The Dart side
/// calls `hamsa/launcher.launchUrl`.
class MainActivity : FlutterActivity() {
    private val channelName = "hamsa/launcher"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
            .setMethodCallHandler { call, result ->
                if (call.method == "launchUrl") {
                    val url = call.argument<String>("url")
                    if (url.isNullOrBlank()) {
                        result.error("BAD_URL", "url is required", null)
                        return@setMethodCallHandler
                    }
                    try {
                        startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("LAUNCH_FAILED", e.message, null)
                    }
                } else {
                    result.notImplemented()
                }
            }
    }
}
