import 'dart:convert';

import 'package:dio/dio.dart';

/// Minimal mocked-Dio harness shared by the US5 (T031) widget tests: records
/// every outgoing request and answers from a method+path stub table.
class StubAdapter implements HttpClientAdapter {
  StubAdapter(this._routes);

  /// `'(METHOD, /path)'` → `(status, json body)` — `''` body for 204s.
  final Map<String, (int, String)> _routes;

  final requests = <StubRequest>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<List<int>>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final raw = options.data;
    final body = switch (raw) {
      final String s when s.isNotEmpty => jsonDecode(s),
      final Map m => m,
      _ => null,
    };
    requests.add(StubRequest(options.method, options.path, body));
    if (requestStream != null) {
      await requestStream.drain<void>();
    }
    final (status, data) =
        _routes['${options.method} ${options.path}'] ?? (404, '{}');
    return ResponseBody.fromString(
      data,
      status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

class StubRequest {
  const StubRequest(this.method, this.path, this.body);

  final String method;
  final String path;

  /// Decoded JSON request body (null when the call carried none).
  final Object? body;
}

Dio stubDio(StubAdapter adapter) =>
    Dio(BaseOptions(baseUrl: 'https://stub.test'))..httpClientAdapter = adapter;
