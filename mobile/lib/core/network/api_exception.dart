import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'package:hamsa/core/l10n/app_localizations.dart';

/// One validation failure detail from the error envelope
/// (`{"field": ..., "rule": ...}` — contracts/api.md).
class FieldRule {
  const FieldRule({required this.field, required this.rule});

  factory FieldRule.fromJson(Map<String, dynamic> json) => FieldRule(
        field: json['field']?.toString() ?? '',
        rule: json['rule']?.toString() ?? '',
      );

  final String field;
  final String rule;
}

/// Normalized API failure. [code] is the locale-neutral envelope code (or a
/// client-side transport token like `network`/`timeout`); Persian copy is
/// resolved at display time via [apiErrorMessage].
class ApiException implements Exception {
  ApiException({
    required this.code,
    this.serverMessage,
    this.statusCode,
    this.details = const [],
  });

  /// Parses any [DioException] into an [ApiException], reading the standard
  /// `{"error": {"code","message","details"}}` envelope when present.
  factory ApiException.from(DioException error) {
    final response = error.response;
    if (response != null) {
      final data = response.data;
      final envelope = data is Map<String, dynamic>
          ? data['error']
          : null;
      if (envelope is Map<String, dynamic>) {
        return ApiException(
          code: envelope['code']?.toString() ?? _statusCodeCode(response.statusCode),
          serverMessage: envelope['message']?.toString(),
          statusCode: response.statusCode,
          details: (envelope['details'] as List<dynamic>? ?? [])
              .whereType<Map<String, dynamic>>()
              .map(FieldRule.fromJson)
              .toList(),
        );
      }
      return ApiException(
        code: _statusCodeCode(response.statusCode),
        statusCode: response.statusCode,
      );
    }

    return switch (error.type) {
      DioExceptionType.connectionTimeout ||
      DioExceptionType.sendTimeout ||
      DioExceptionType.receiveTimeout =>
        ApiException(code: 'timeout'),
      DioExceptionType.connectionError => ApiException(code: 'network'),
      DioExceptionType.cancel => ApiException(code: 'cancelled'),
      _ => ApiException(code: 'unknown'),
    };
  }

  static String _statusCodeCode(int? status) => switch (status) {
        400 => 'VALIDATION_ERROR',
        401 => 'UNAUTHENTICATED',
        403 => 'FORBIDDEN',
        404 => 'NOT_FOUND',
        409 => 'CONFLICT',
        429 => 'RATE_LIMITED',
        _ => status != null && status >= 500 ? 'INTERNAL' : 'unknown',
      };

  final String code;

  /// Server-provided Persian message, when the envelope carried one.
  final String? serverMessage;

  final int? statusCode;
  final List<FieldRule> details;

  bool get isUnauthenticated => code == 'UNAUTHENTICATED';

  @override
  String toString() => 'ApiException($code): $serverMessage';
}

/// Resolves the user-facing Persian message for an [ApiException].
extension ApiErrorL10n on AppLocalizations {
  String apiErrorMessage(ApiException e) => switch (e.code) {
        'network' => errorNetwork,
        'timeout' => errorTimeout,
        'cancelled' => errorUnknown,
        'UNAUTHENTICATED' => errorUnauthorized,
        'FORBIDDEN' => errorForbidden,
        'NOT_FOUND' => errorNotFound,
        'CONFLICT' => errorConflict,
        'RATE_LIMITED' => errorRateLimited,
        'VALIDATION_ERROR' => errorValidation,
        'INTERNAL' => errorServer,
        _ => e.serverMessage ?? errorUnknown,
      };
}

/// Maps ANY error thrown by an API call to a displayable Persian message.
/// `DioException`s (backend unreachable, timeouts, …) are parsed via
/// [ApiException.from]; connection failures always carry a diagnostics line
/// with the failure type and target host:port so "nothing happened" cases
/// (e.g. the phone cannot reach the backend) are visible on-device.
String describeError(AppLocalizations l10n, Object error) {
  if (error is ApiException) return l10n.apiErrorMessage(error);
  if (error is DioException) {
    var message = l10n.apiErrorMessage(ApiException.from(error));
    if (error.response == null) {
      final uri = error.requestOptions.uri;
      message += '\n[diagnostic] ${error.type.name} → ${uri.host}:${uri.port}';
    }
    return message;
  }
  if (kDebugMode) return '[diagnostic] ${error.runtimeType}: $error';
  return l10n.errorUnknown;
}
