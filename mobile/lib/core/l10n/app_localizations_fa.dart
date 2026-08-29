// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Persian (`fa`).
class AppLocalizationsFa extends AppLocalizations {
  AppLocalizationsFa([String locale = 'fa']) : super(locale);

  @override
  String get appTitle => 'همسا';

  @override
  String get splashMessage => 'در حال بارگذاری…';

  @override
  String get roleManager => 'مدیر';

  @override
  String get roleResident => 'ساکن';

  @override
  String get loginTitle => 'ورود به همسا';

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
  String resendIn(String seconds) {
    return 'ارسال مجدد کد تا $seconds ثانیه دیگر';
  }

  @override
  String otpSentTo(String phone) {
    return 'کد تأیید به شماره $phone پیامک شد.';
  }

  @override
  String get resendOtp => 'ارسال مجدد کد';

  @override
  String get changePhone => 'تغییر شماره';

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

  @override
  String get buildingsTitle => 'ساختمان‌ها';

  @override
  String get buildingName => 'نام ساختمان';

  @override
  String get buildingAddress => 'نشانی';

  @override
  String get blockCount => 'تعداد بلوک';

  @override
  String get floorCount => 'تعداد طبقات';

  @override
  String get unitCountLabel => 'تعداد واحدها';

  @override
  String get builtYear => 'سال ساخت';

  @override
  String get managerPhoneLabel => 'شماره مدیر';

  @override
  String get emergencyPhoneLabel => 'شماره اضطراری';

  @override
  String get notesLabel => 'یادداشت';

  @override
  String get addBuilding => 'افزودن ساختمان';

  @override
  String get editBuilding => 'ویرایش ساختمان';

  @override
  String get archiveBuilding => 'آرشیو ساختمان';

  @override
  String get archiveConfirm => 'آیا از آرشیو این مورد مطمئن هستید؟';

  @override
  String get unitsTitle => 'واحدها';

  @override
  String get unitNumber => 'شماره واحد';

  @override
  String get unitBlock => 'بلوک';

  @override
  String get unitFloor => 'طبقه';

  @override
  String get areaM2 => 'متراژ (متر مربع)';

  @override
  String get parkingCount => 'تعداد پارکینگ';

  @override
  String get parkingNumbers => 'شماره پارکینگ‌ها';

  @override
  String get storageCount => 'تعداد انباری';

  @override
  String get storageNumbers => 'شماره انباری‌ها';

  @override
  String get unitStatus => 'وضعیت واحد';

  @override
  String get statusActive => 'فعال';

  @override
  String get statusVacant => 'خالی';

  @override
  String get statusOccupied => 'مسکونی';

  @override
  String get statusInactive => 'غیرفعال';

  @override
  String get addUnit => 'افزودن واحد';

  @override
  String get editUnit => 'ویرایش واحد';

  @override
  String get searchUnits => 'جست‌وجوی واحد…';

  @override
  String get allBlocks => 'همه بلوک‌ها';

  @override
  String get allStatuses => 'همه وضعیت‌ها';

  @override
  String get changeHistory => 'تاریخچه تغییرات';

  @override
  String get noChangeHistory => 'تغییری ثبت نشده است.';

  @override
  String get invalidFloor => 'طبقه را صحیح وارد کنید.';

  @override
  String get invalidNumber => 'عدد صحیح وارد کنید.';

  @override
  String get peopleTitle => 'افراد و ساکنان';

  @override
  String get addPerson => 'افزودن شخص';

  @override
  String get editPerson => 'ویرایش شخص';

  @override
  String get personName => 'نام و نام خانوادگی';

  @override
  String get personPhone => 'شماره موبایل';

  @override
  String get personNationalId => 'کد ملی';

  @override
  String get invalidNationalId => 'کد ملی باید ۱۰ رقم باشد.';

  @override
  String get relationshipLabel => 'نسبت ساکن';

  @override
  String get relOwner => 'مالک';

  @override
  String get relTenant => 'مستأجر';

  @override
  String get relNonResidentOwner => 'مالک غیرمقیم';

  @override
  String get selectPerson => 'انتخاب شخص';

  @override
  String get addOccupancy => 'ثبت سکونت';

  @override
  String get occupancyTitle => 'ساکنان و سابقه سکونت';

  @override
  String get occupancyStart => 'تاریخ شروع سکونت';

  @override
  String get occupancyActive => 'فعال';

  @override
  String get occupancyEnded => 'پایان‌یافته';

  @override
  String get endOccupancy => 'پایان سکونت';

  @override
  String get occupantCountTitle => 'تعداد ساکن';

  @override
  String get recordOccupantCount => 'ثبت تعداد جدید';

  @override
  String get effectiveFrom => 'تاریخ اثر';

  @override
  String get occupancyNeedsPerson =>
      'هنوز شخصی برای این ساختمان ثبت نشده است؛ ابتدا یک شخص اضافه کنید.';

  @override
  String get billingTitle => 'شارژها';

  @override
  String get addPeriod => 'دوره جدید';

  @override
  String get editPeriod => 'ویرایش دوره';

  @override
  String get periodTitleLabel => 'عنوان دوره';

  @override
  String get periodStart => 'تاریخ شروع';

  @override
  String get periodEnd => 'تاریخ پایان';

  @override
  String get periodDue => 'تاریخ سررسید';

  @override
  String get lateFeeType => 'نوع دیرکرد';

  @override
  String get lateFeeNone => 'بدون دیرکرد';

  @override
  String get lateFeeFixed => 'مبلغ ثابت';

  @override
  String get lateFeePercent => 'درصدی';

  @override
  String get lateFeePerDay => 'روزانه';

  @override
  String get lateFeeValue => 'مقدار دیرکرد';

  @override
  String get costItemsTitle => 'اقلام هزینه';

  @override
  String get addCostItem => 'افزودن قلم هزینه';

  @override
  String get editCostItem => 'ویرایش قلم هزینه';

  @override
  String get costItemTitle => 'عنوان قلم';

  @override
  String get costItemAmount => 'مبلغ';

  @override
  String get calcMethod => 'روش محاسبه';

  @override
  String get methodEqual => 'مساوی';

  @override
  String get methodPerOccupant => 'سرشمار (ساکن)';

  @override
  String get methodPerArea => 'متراژ';

  @override
  String get methodFixed => 'مبلغ ثابت';

  @override
  String get methodSpecificUnits => 'واحدهای مشخص';

  @override
  String get methodCombined => 'ترکیبی';

  @override
  String get fixedPerUnit => 'مبلغ هر واحد';

  @override
  String get includeVacant => 'شامل واحدهای خالی';

  @override
  String get selectUnits => 'انتخاب واحدها';

  @override
  String get comboWeights => 'ترکیب وزن‌ها';

  @override
  String get addWeight => 'افزودن سهم';

  @override
  String get comboWeightsInvalid => 'جمع وزن‌ها باید ۱۰۰ باشد';

  @override
  String get calculate => 'محاسبه';

  @override
  String get recalculate => 'محاسبه مجدد';

  @override
  String get previewTitle => 'پیش‌نمایش محاسبه';

  @override
  String get reconciliationOk => 'مغایرت ندارد';

  @override
  String get reconciliationBad => 'مغایرت در جمع اقلام!';

  @override
  String get exactShare => 'سهم دقیق';

  @override
  String get issue => 'صدور صورتحساب‌ها';

  @override
  String get issueConfirm =>
      'پس از صدور، صورتحساب‌ها تغییرناپذیرند و ساکنان مطلع می‌شوند. ادامه می‌دهید؟';

  @override
  String get issuedOk => 'صورتحساب‌ها صادر شد';

  @override
  String get reopen => 'بازگشایی دوره';

  @override
  String get reopenConfirm =>
      'پیش‌نمایش محاسبه حذف و دوره به پیش‌نویس بازمی‌گردد. ادامه می‌دهید؟';

  @override
  String get closePeriod => 'بستن دوره';

  @override
  String get closeConfirm =>
      'با بستن دوره دیگر هیچ تغییری امکان‌پذیر نیست. ادامه می‌دهید؟';

  @override
  String get invoicesTitle => 'صورتحساب‌ها';

  @override
  String get baseAmount => 'شارژ دوره';

  @override
  String get priorDebt => 'بدهی قبلی';

  @override
  String get lateFeeAmount => 'دیرکرد';

  @override
  String get creditAmount => 'اعتبار';

  @override
  String get finalAmount => 'مبلغ نهایی';

  @override
  String get paidAmount => 'پرداخت‌شده';

  @override
  String get issueDateLabel => 'تاریخ صدور';

  @override
  String get dueDateLabel => 'تاریخ سررسید';

  @override
  String get itemKindAdjustment => 'اصلاحیه';

  @override
  String get adjustmentDebit => 'بدهکار';

  @override
  String get adjustmentCredit => 'بستانکار';

  @override
  String get addAdjustment => 'ثبت اصلاحیه';

  @override
  String get adjustmentReason => 'دلیل اصلاحیه';

  @override
  String get cancelInvoice => 'ابطال صورتحساب';

  @override
  String get cancelInvoiceConfirm =>
      'صورتحساب ابطال شود؟ ردیف و سابقه آن حفظ خواهد شد.';

  @override
  String get cancelReason => 'دلیل ابطال';

  @override
  String get myCharges => 'شارژهای من';

  @override
  String get paymentsTitle => 'پرداخت‌ها';

  @override
  String get recordPayment => 'ثبت پرداخت';

  @override
  String get paymentAmountLabel => 'مبلغ پرداخت (تومان)';

  @override
  String get paymentDateLabel => 'تاریخ پرداخت';

  @override
  String get trackingNumberLabel => 'شماره پیگیری / رسید (اختیاری)';

  @override
  String get paymentRecordedOk => 'پرداخت با موفقیت ثبت شد';

  @override
  String get ledgerTitle => 'دفتر پرداخت‌ها';

  @override
  String get filterMethodLabel => 'روش پرداخت';

  @override
  String get filterAll => 'همه';

  @override
  String get filterFromDate => 'از تاریخ';

  @override
  String get filterToDate => 'تا تاریخ';

  @override
  String get payTitle => 'پرداخت آنلاین';

  @override
  String get outstandingLabel => 'مبلغ باقی‌مانده';

  @override
  String get payLaunchConfirm =>
      'به درگاه پرداخت منتقل می‌شوید. پس از بازگشت، وضعیت پرداخت بررسی می‌شود.';

  @override
  String get paySuccess => 'پرداخت با موفقیت انجام شد';

  @override
  String get payFailed => 'پرداخت ناموفق بود یا تأیید نشد';

  @override
  String get launchFailed => 'باز کردن درگاه پرداخت ممکن نشد';

  @override
  String get receiptTitle => 'رسید پرداخت';

  @override
  String get paymentHistoryTitle => 'تاریخچه پرداخت‌ها';

  @override
  String get unitLabel => 'واحد';

  @override
  String get payNow => 'پرداخت آنلاین';

  @override
  String get noPayments => 'هنوز پرداختی ثبت نشده است';

  @override
  String get expensesTitle => 'هزینه‌ها';

  @override
  String get financialReport => 'گزارش مالی';

  @override
  String get addExpense => 'ثبت هزینه';

  @override
  String get editExpense => 'ویرایش هزینه';

  @override
  String get expenseTitleLabel => 'عنوان هزینه';

  @override
  String get expenseCategoryLabel => 'دسته‌بندی';

  @override
  String get expenseAmountLabel => 'مبلغ (تومان)';

  @override
  String get expenseDateLabel => 'تاریخ هزینه';

  @override
  String get expenseApprovalLabel => 'وضعیت تأیید';

  @override
  String get expenseDescriptionLabel => 'توضیحات (اختیاری)';

  @override
  String get filterCategoryLabel => 'دسته‌بندی';

  @override
  String get filterApprovalLabel => 'وضعیت تأیید';

  @override
  String get noExpenses => 'هنوز هزینه‌ای ثبت نشده است';

  @override
  String get expenseSavedOk => 'هزینه با موفقیت ثبت شد';

  @override
  String get expenseDeletedOk => 'هزینه حذف شد';

  @override
  String get deleteExpense => 'حذف هزینه';

  @override
  String get deleteExpenseConfirm =>
      'این هزینه حذف شود؟ ردیف آن از گزارش‌ها حذف می‌شود ولی سابقه مالی حفظ می‌شود.';

  @override
  String get reportMonthLabel => 'انتخاب ماه';

  @override
  String get reportMonthlyIncome => 'درآمد این ماه';

  @override
  String get reportMonthlyExpense => 'هزینه این ماه';

  @override
  String get reportNet => 'تراز این ماه';

  @override
  String get reportTotalDebt => 'مجموع بدهی ساکنان';

  @override
  String get reportTotalPayments => 'مجموع پرداخت‌ها';

  @override
  String get reportTotalExpenses => 'مجموع هزینه‌ها';
}
