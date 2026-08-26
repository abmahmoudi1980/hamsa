// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Persian (`fa`).
class AppLocalizationsFa extends AppLocalizations {
  AppLocalizationsFa([String locale = 'fa']) : super(locale);

  @override
  String get appTitle => 'هامسا';

  @override
  String get splashMessage => 'در حال بارگذاری…';

  @override
  String get roleManager => 'مدیر';

  @override
  String get roleResident => 'ساکن';

  @override
  String get loginTitle => 'ورود به هامسا';

  @override
  String get loginSubtitle => 'برای ورود، شماره موبایل خود را وارد کنید.';

  @override
  String get phoneLabel => 'شماره موبایل';

  @override
  String get requestOtp => 'دریافت کد تأیید';

  @override
  String get otpLabel => 'کد تأیید پیامک‌شده';

  @override
  String get verifyAndLogin => 'تأیید و ورود';

  @override
  String resendIn(int seconds) {
    return 'ارسال مجدد کد تا $seconds ثانیه دیگر';
  }

  @override
  String get invalidMobile => 'شماره موبایل معتبر نیست.';

  @override
  String get invalidOtp => 'کد تأیید را وارد کنید.';

  @override
  String get errorNetwork =>
      'خطا در اتصال به سرور؛ اتصال اینترنت را بررسی کنید.';

  @override
  String get errorTimeout =>
      'زمان انتظار پاسخ سرور به پایان رسید؛ دوباره تلاش کنید.';

  @override
  String get errorUnauthorized => 'احراز هویت لازم است.';

  @override
  String get errorForbidden => 'دسترسی غیرمجاز است.';

  @override
  String get errorNotFound => 'مورد درخواستی یافت نشد.';

  @override
  String get errorConflict => 'وضعیت ثبت با وضعیت فعلی هم‌خوانی ندارد.';

  @override
  String get errorRateLimited =>
      'تعداد درخواست‌ها بیش از حد مجاز است؛ کمی صبر کنید.';

  @override
  String get requiredField => 'این فیلد الزامی است.';

  @override
  String get invalidAmount => 'مبلغ را به‌صورت عدد بزرگ‌تر از صفر وارد کنید.';

  @override
  String get errorValidation => 'اطلاعات ارسالی معتبر نیست.';

  @override
  String get errorServer => 'خطای داخلی سرور؛ لطفاً دوباره تلاش کنید.';

  @override
  String get errorUnknown => 'خطایی رخ داد؛ دوباره تلاش کنید.';

  @override
  String get emptyStateTitle => 'هنوز موردی ثبت نشده است';

  @override
  String get emptyStateSubtitle => 'برای شروع، اولین مورد را اضافه کنید.';

  @override
  String get retry => 'تلاش دوباره';

  @override
  String get cancel => 'انصراف';

  @override
  String get confirm => 'تأیید';

  @override
  String get save => 'ذخیره';

  @override
  String get edit => 'ویرایش';

  @override
  String get delete => 'حذف';

  @override
  String get logout => 'خروج از حساب';

  @override
  String get selectDate => 'انتخاب تاریخ';

  @override
  String get tomanSuffix => 'تومان';

  @override
  String get attachImage => 'افزودن تصویر';

  @override
  String get removeAttachment => 'حذف تصویر';

  @override
  String get notificationsTitle => 'اعلان‌ها';

  @override
  String get markAllRead => 'علامت‌گذاری همه به‌عنوان خوانده‌شده';

  @override
  String unreadBadge(int count) {
    return '$count ناخواندا';
  }

  @override
  String get managerShellTitle => 'پنل مدیریت';

  @override
  String get residentShellTitle => 'پنل ساکن';

  @override
  String get shellUnderConstruction =>
      'این بخش در مراحل بعدی پیاده‌سازی می‌شود.';
}
