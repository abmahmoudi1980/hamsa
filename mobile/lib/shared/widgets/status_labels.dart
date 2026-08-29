import 'package:flutter/material.dart';

/// Persian labels for wire enums (contracts/api.md: enum values stay
/// locale-neutral tokens; the client maps them to Persian labels).
///
/// Unknown values fall back to the raw token so new server-side values never
/// render blank.

const Map<String, String> invoiceStatusLabels = {
  'unpaid': 'پرداخت‌نشده',
  'partial': 'پرداخت جزئی',
  'paid': 'پرداخت‌شده',
  'expired': 'منقضی‌شده',
  'cancelled': 'لغوشده',
};

const Map<String, Color> invoiceStatusColors = {
  'unpaid': Color(0xFFE53935),
  'partial': Color(0xFFFFB300),
  'paid': Color(0xFF43A047),
  'expired': Color(0xFF8D6E63),
  'cancelled': Color(0xFF757575),
};

const Map<String, String> periodStatusLabels = {
  'draft': 'پیش‌نویس',
  'calculated': 'محاسبه‌شده',
  'issued': 'صادره',
  'closed': 'بسته',
};

const Map<String, String> maintenanceStatusLabels = {
  'new': 'ثبت‌شده',
  'under_review': 'در حال بررسی',
  'in_progress': 'در حال انجام',
  'done': 'انجام‌شده',
  'closed': 'بسته‌شده',
};

const Map<String, Color> maintenanceStatusColors = {
  'new': Color(0xFF1E88E5),
  'under_review': Color(0xFF8E24AA),
  'in_progress': Color(0xFFFB8C00),
  'done': Color(0xFF43A047),
  'closed': Color(0xFF757575),
};

/// US5 payment status/method labels (migration 0005 enums).
const Map<String, String> paymentStatusLabels = {
  'recorded': 'در انتظار تأیید',
  'verified': 'تأییدشده',
  'failed': 'ناموفق',
  'reversed': 'برگشتی',
};

const Map<String, Color> paymentStatusColors = {
  'recorded': Color(0xFFFB8C00),
  'verified': Color(0xFF43A047),
  'failed': Color(0xFFE53935),
  'reversed': Color(0xFF757575),
};

const Map<String, String> paymentMethodLabels = {
  'manual': 'دستی',
  'gateway': 'درگاه اینترنتی',
};
