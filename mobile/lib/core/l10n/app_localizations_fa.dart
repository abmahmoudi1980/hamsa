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
  String get loginSubtitle =>
      'برای ورود، شماره موبایل و رمز عبور خود را وارد کنید.';

  @override
  String get phoneLabel => 'شماره موبایل';

  @override
  String get loginButton => 'ورود';

  @override
  String get passwordLabel => 'رمز عبور';

  @override
  String get registerTitle => 'ثبت‌نام ساکن';

  @override
  String get registerSubtitle => 'با کد دعوت مدیر ساختمان ثبت‌نام کنید.';

  @override
  String get registerButton => 'ثبت‌نام';

  @override
  String get gotoRegister => 'حساب ندارید؟ ثبت‌نام با کد دعوت';

  @override
  String get gotoSetup => 'راه‌اندازی اولیه ساختمان (مدیر)';

  @override
  String get inviteCodeLabel => 'کد دعوت';

  @override
  String get nameLabel => 'نام و نام خانوادگی (اختیاری)';

  @override
  String get setupTitle => 'راه‌اندازی اولیه';

  @override
  String get setupSubtitle => 'اولین حساب مدیریت ساختمان را بسازید.';

  @override
  String get setupButton => 'ساخت حساب مدیر';

  @override
  String get confirmPasswordLabel => 'تکرار رمز عبور';

  @override
  String get passwordsMismatch => 'رمزها یکسان نیستند.';

  @override
  String get weakPassword =>
      'رمز عبور باید حداقل ۸ کاراکتر و شامل حرف و عدد باشد.';

  @override
  String get invalidInviteCode => 'کد دعوت را وارد کنید.';

  @override
  String get inviteTitle => 'دعوت ساکن';

  @override
  String get inviteDescription =>
      'برای هر ساکن یک کد دعوت یکتا صادر می‌شود؛ کد را به او بدهید تا در اپ ثبت‌نام کند. هر کد ۷ روز اعتبار دارد و فقط یک‌بار قابل استفاده است.';

  @override
  String get getInviteCode => 'دریافت کد دعوت';

  @override
  String get inviteCopied => 'کد دعوت کپی شد';

  @override
  String get invalidMobile => 'شماره موبایل معتبر نیست.';

  @override
  String get invalidCode => 'کد دعوت نامعتبر است.';

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

  @override
  String inviteExpiresInDays(int days) {
    return 'اعتبار کد: $days روز';
  }

  @override
  String get maintenanceTitle => 'درخواست‌های نگهداری';

  @override
  String get myMaintenanceTitle => 'درخواست‌های من';

  @override
  String get addMaintenanceRequest => 'ثبت درخواست';

  @override
  String get newMaintenanceTitle => 'ثبت درخواست جدید';

  @override
  String get maintenanceDetailTitle => 'جزئیات درخواست';

  @override
  String get maintenanceTitleLabel => 'عنوان درخواست';

  @override
  String get maintenanceTitleHint => 'مثلاً: آسانسور لرزش دارد';

  @override
  String get maintenanceTitleRequired => 'عنوان الزامی است';

  @override
  String get maintenanceCategoryLabel => 'دسته‌بندی';

  @override
  String get maintenancePriorityLabel => 'اولویت';

  @override
  String get maintenanceLocationLabel => 'محل (اختیاری)';

  @override
  String get maintenanceLocationHint => 'مثلاً: طبقه ۳، آسانسور';

  @override
  String get maintenanceDescriptionLabel => 'توضیحات (اختیاری)';

  @override
  String get maintenancePhotoLabel => 'تصویر (اختیاری)';

  @override
  String get maintenanceSubmitSuccess => 'درخواست با موفقیت ثبت شد';

  @override
  String get maintenanceSubmitError => 'خطا در ثبت درخواست';

  @override
  String get maintenanceLoadError => 'خطا در دریافت';

  @override
  String get maintenanceEmptyTitle => 'هنوز درخواستی ثبت نکرده‌اید';

  @override
  String get maintenanceEmptySubtitle =>
      'برای ثبت مشکل جدید، دکمه «ثبت درخواست» را بزنید.';

  @override
  String get maintenanceNoRequests => 'درخواستی یافت نشد';

  @override
  String get maintenanceNoRequestsSubtitle =>
      'با فیلترهای فعلی هیچ درخواستی وجود ندارد.';

  @override
  String get maintenanceFilterAllStatuses => 'همه وضعیت‌ها';

  @override
  String get maintenanceFilterAllPriorities => 'همه اولویت‌ها';

  @override
  String get maintenanceFilterAllCategories => 'همه دسته‌ها';

  @override
  String maintenanceCreatedAt(String date) {
    return 'تاریخ ثبت: $date';
  }

  @override
  String maintenanceLocation(String location) {
    return 'محل: $location';
  }

  @override
  String maintenanceRecordedCost(String amount) {
    return 'هزینه ثبت‌شده: $amount تومان';
  }

  @override
  String get maintenanceChangeStatus => 'تغییر وضعیت';

  @override
  String maintenanceChangeStatusConfirm(String status) {
    return 'وضعیت درخواست به «$status» تغییر کند؟';
  }

  @override
  String maintenanceStatusChangedTo(String status) {
    return 'وضعیت به «$status» تغییر کرد';
  }

  @override
  String get maintenanceStatusChangeNotAllowed => 'تغییر وضعیت مجاز نیست';

  @override
  String get maintenanceMetaTitle => 'مسئول و هزینه';

  @override
  String get maintenanceAssigneeLabel => 'شناسه مسئول (UUID)';

  @override
  String get maintenanceAssigneeHint => 'اختیاری';

  @override
  String get maintenanceCostLabel => 'هزینه ثبت‌شده (تومان)';

  @override
  String get maintenanceCostHint => 'مثلاً ۵۰۰۰۰۰';

  @override
  String get maintenanceNotesLabel => 'یادداشت مدیر';

  @override
  String get maintenanceSaveMeta => 'ذخیره مسئول / هزینه / یادداشت';

  @override
  String get maintenanceSaved => 'تغییرات ذخیره شد';

  @override
  String get maintenanceSaveError => 'خطا در ذخیره';

  @override
  String get maintenanceBack => 'بازگشت';

  @override
  String get maintenanceClosedChip => 'این درخواست بسته شده است';

  @override
  String get maintenanceStatusNew => 'ثبت‌شده';

  @override
  String get maintenanceStatusUnderReview => 'در حال بررسی';

  @override
  String get maintenanceStatusInProgress => 'در حال انجام';

  @override
  String get maintenanceStatusDone => 'انجام‌شده';

  @override
  String get maintenanceStatusClosed => 'بسته‌شده';

  @override
  String get maintenanceCategoryElevator => 'آسانسور';

  @override
  String get maintenanceCategoryUtilities => 'تأسیسات';

  @override
  String get maintenanceCategoryElectrical => 'برق';

  @override
  String get maintenanceCategoryWater => 'آب';

  @override
  String get maintenanceCategoryCleaning => 'نظافت';

  @override
  String get maintenanceCategoryCommonArea => 'مشاعات';

  @override
  String get maintenanceCategoryParking => 'پارکینگ';

  @override
  String get maintenanceCategoryOther => 'سایر';

  @override
  String get maintenancePriorityNormal => 'عادی';

  @override
  String get maintenancePriorityImportant => 'مهم';

  @override
  String get maintenancePriorityUrgent => 'فوری';

  @override
  String get maintenanceResidentMenu => 'درخواست‌های نگهداری';

  @override
  String get announcementsTitle => 'اطلاعیه‌ها';

  @override
  String get myAnnouncementsTitle => 'اطلاعیه‌های من';

  @override
  String get addAnnouncement => 'انتشار اطلاعیه';

  @override
  String get editAnnouncement => 'ویرایش اطلاعیه';

  @override
  String get announcementTitleLabel => 'عنوان اطلاعیه';

  @override
  String get announcementTitleHint => 'مثلاً: جلسه هیئت مدیره';

  @override
  String get announcementTitleRequired => 'عنوان الزامی است';

  @override
  String get announcementBodyLabel => 'متن اطلاعیه';

  @override
  String get announcementBodyRequired => 'متن اطلاعیه الزامی است';

  @override
  String get announcementAudienceLabel => 'مخاطب';

  @override
  String get announcementAudienceAll => 'همه ساکنان';

  @override
  String get announcementAudienceBlock => 'بلوک';

  @override
  String get announcementAudienceFloor => 'طبقه';

  @override
  String get announcementAudienceUnit => 'واحد مشخص';

  @override
  String get announcementAudienceValueLabel => 'مقدار مخاطب';

  @override
  String get announcementAudienceValueHint => 'مثلاً: A یا ۲ یا شماره واحد';

  @override
  String get announcementPublishAtLabel => 'تاریخ انتشار (اختیاری)';

  @override
  String get announcementExpireAtLabel => 'تاریخ انقضا (اختیاری)';

  @override
  String get announcementAttachmentLabel => 'پیوست (اختیاری)';

  @override
  String get announcementSubmitSuccess => 'اطلاعیه با موفقیت منتشر شد';

  @override
  String get announcementSubmitError => 'خطا در انتشار اطلاعیه';

  @override
  String get announcementUpdateSuccess => 'اطلاعیه ویرایش شد';

  @override
  String get announcementDeleteConfirm => 'این اطلاعیه حذف شود؟';

  @override
  String get announcementDeleted => 'اطلاعیه حذف شد';

  @override
  String get announcementLoadError => 'خطا در دریافت اطلاعیه‌ها';

  @override
  String get announcementEmptyTitle => 'هنوز اطلاعیه‌ای منتشر نشده است';

  @override
  String get announcementEmptySubtitle =>
      'برای اطلاع‌رسانی جدید، «انتشار اطلاعیه» را بزنید.';

  @override
  String get announcementResidentEmptyTitle =>
      'هنوز اطلاعیه‌ای برای شما منتشر نشده است';

  @override
  String get announcementResidentEmptySubtitle =>
      'اطلاعیه‌های جدید اینجا نمایش داده می‌شود.';

  @override
  String get announcementDetailTitle => 'جزئیات اطلاعیه';

  @override
  String get announcementUnread => 'نخوانده';

  @override
  String get announcementRead => 'خوانده‌شده';

  @override
  String get announcementMarkRead => 'علامت‌گذاری به‌عنوان خوانده‌شده';

  @override
  String get announcementMarkedRead => 'به‌عنوان خوانده‌شده ثبت شد';

  @override
  String get notificationCenterTitle => 'مرکز اعلان‌ها';

  @override
  String get notificationEmptyTitle => 'اعلانی وجود ندارد';

  @override
  String get notificationEmptySubtitle =>
      'اعلان‌های شارژ، پرداخت و اطلاعیه‌ها اینجا نمایش داده می‌شود.';

  @override
  String get notificationMarkAllRead => 'خواندن همه';

  @override
  String get notificationMarkedAllRead => 'همه اعلان‌ها خوانده شد';

  @override
  String get notificationTypeInvoiceIssued => 'صدور صورتحساب';

  @override
  String get notificationTypeAnnouncementPublished => 'اطلاعیه جدید';

  @override
  String get notificationTypeRequestStatus => 'تغییر وضعیت درخواست';

  @override
  String get announcementAudienceDisplayAll => 'همه';

  @override
  String announcementAudienceDisplayBlock(String value) {
    return 'بلوک $value';
  }

  @override
  String announcementAudienceDisplayFloor(String value) {
    return 'طبقه $value';
  }

  @override
  String announcementAudienceDisplayUnit(String value) {
    return 'واحد $value';
  }

  @override
  String get homeMenu => 'خانه';

  @override
  String get navHome => 'خانه';

  @override
  String get navCharges => 'شارژها';

  @override
  String get navPayments => 'پرداخت‌ها';

  @override
  String get navMaintenance => 'تعمیرات';

  @override
  String get navAnnouncements => 'اطلاعیه‌ها';

  @override
  String get navProfile => 'پروفایل';

  @override
  String get homeTitle => 'خانه';

  @override
  String get homeFinancialCard => 'وضعیت مالی';

  @override
  String get homePayableAmount => 'مبلغ قابل پرداخت';

  @override
  String get homeNoInvoice => 'صورتحساب فعالی ندارید';

  @override
  String get homeLatestInvoice => 'آخرین صورتحساب';

  @override
  String homeDueDate(String date) {
    return 'سررسید: $date';
  }

  @override
  String get homeRequestsCard => 'درخواست‌های نگهداری';

  @override
  String homeOpenRequests(int count) {
    return '$count درخواست باز';
  }

  @override
  String get homeLatestRequest => 'آخرین درخواست';

  @override
  String get homeNoOpenRequest => 'درخواست بازی ثبت نشده است';

  @override
  String get homeAnnouncementsCard => 'اطلاعیه‌ها';

  @override
  String homeUnreadAnnouncements(int count) {
    return '$count ناخوانده';
  }

  @override
  String get homeViewAll => 'مشاهده همه';

  @override
  String get homeViewCharges => 'مشاهده شارژها';

  @override
  String get homeViewAnnouncements => 'مشاهده اطلاعیه‌ها';

  @override
  String get homePayInvoice => 'پرداخت صورتحساب';

  @override
  String get profileTitle => 'پروفایل';

  @override
  String get profilePhone => 'شماره موبایل';

  @override
  String get profileName => 'نام و نام خانوادگی';

  @override
  String get profileEditName => 'ویرایش نام';

  @override
  String get profileSave => 'ذخیره';

  @override
  String get profileSavedOk => 'نام شما ذخیره شد';

  @override
  String get profileError => 'خطا در ذخیره نام';

  @override
  String get dashboardTitle => 'داشبورد';

  @override
  String get dashboardCards => 'شاخص‌ها';

  @override
  String get dashboardUnitCount => 'تعداد واحدها';

  @override
  String get dashboardOccupiedUnits => 'واحدهای مسکونی';

  @override
  String get dashboardDebtorUnits => 'واحدهای بدهکار';

  @override
  String get dashboardTotalDebt => 'مجموع بدهی';

  @override
  String get dashboardMonthIncome => 'درآمد ماه';

  @override
  String get dashboardMonthExpense => 'هزینه ماه';

  @override
  String get dashboardOpenRequests => 'درخواست‌های باز';

  @override
  String get dashboardPendingExpenses => 'هزینه‌های در انتظار تأیید';

  @override
  String get dashboardAlerts => 'هشدارها';

  @override
  String get dashboardQuickActions => 'اقدام‌های سریع';

  @override
  String dashboardAlertDebtor(String unit) {
    return 'واحد $unit بدهکار است';
  }

  @override
  String dashboardAlertPastDue(String unit) {
    return 'صورتحساب واحد $unit سررسید گذشته';
  }

  @override
  String dashboardAlertOpenRequest(String title) {
    return 'درخواست باز: $title';
  }

  @override
  String dashboardAlertPendingExpense(String title) {
    return 'هزینه در انتظار تأیید: $title';
  }

  @override
  String get dashboardActionIssueCharge => 'صدور شارژ';

  @override
  String get dashboardActionRecordExpense => 'ثبت هزینه';

  @override
  String get dashboardActionRecordPayment => 'ثبت پرداخت';

  @override
  String get dashboardActionSendAnnouncement => 'انتشار اطلاعیه';

  @override
  String get dashboardActionOpenRequests => 'درخواست‌ها';

  @override
  String get dashboardNoAlerts => 'هشدار فعالی وجود ندارد';

  @override
  String get dashboardBuildingPicker => 'انتخاب ساختمان';

  @override
  String get dashboardNoBuilding => 'ابتدا یک ساختمان ایجاد کنید.';

  @override
  String get roleSuperAdmin => 'سرپرست ارشد';

  @override
  String get superadminHomeTitle => 'پنل سرپرست ارشد';

  @override
  String get superadminHomeHint =>
      'ساختمانی برای مدیریت ندارید؛ برای شروع، مدیر تازه دعوت کنید.';

  @override
  String inviteIssuedRole(String role) {
    return 'کد دعوت با نقش «$role» صادر شد.';
  }

  @override
  String get buildingManagersTitle => 'مدیران ساختمان';

  @override
  String get addManager => 'افزودن مدیر';

  @override
  String get managerAdded => 'مدیر با موفقیت اضافه شد.';

  @override
  String get managerRemoved => 'دسترسی مدیر با موفقیت حذف شد.';

  @override
  String get removeManagerTitle => 'حذف دسترسی مدیر';

  @override
  String get removeManagerBody =>
      'آیا مطمئنید می‌خواهید دسترسی این مدیر از ساختمان حذف شود؟';

  @override
  String get managersEmpty => 'هنوز مدیری برای این ساختمان ثبت نشده است.';

  @override
  String managersGrantedOn(String date) {
    return 'از تاریخ $date';
  }

  @override
  String get managerAlreadyGrantedError => 'این مدیر از قبل دسترسی دارد.';

  @override
  String get lastManagerError =>
      'حداقل یک مدیر باید برای ساختمان باقی بماند.';

  @override
  String get selfRemoveError =>
      'برای حذف دسترسی خود ابتدا مدیر دیگری معرفی کنید.';
}
