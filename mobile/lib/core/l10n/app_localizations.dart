import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_fa.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[Locale('fa')];

  /// No description provided for @appTitle.
  ///
  /// In fa, this message translates to:
  /// **'همسا'**
  String get appTitle;

  /// No description provided for @splashMessage.
  ///
  /// In fa, this message translates to:
  /// **'در حال بارگذاری…'**
  String get splashMessage;

  /// No description provided for @roleManager.
  ///
  /// In fa, this message translates to:
  /// **'مدیر'**
  String get roleManager;

  /// No description provided for @roleResident.
  ///
  /// In fa, this message translates to:
  /// **'ساکن'**
  String get roleResident;

  /// No description provided for @loginTitle.
  ///
  /// In fa, this message translates to:
  /// **'ورود به همسا'**
  String get loginTitle;

  /// No description provided for @loginSubtitle.
  ///
  /// In fa, this message translates to:
  /// **'برای ورود، شماره موبایل خود را وارد کنید.'**
  String get loginSubtitle;

  /// No description provided for @phoneLabel.
  ///
  /// In fa, this message translates to:
  /// **'شماره موبایل'**
  String get phoneLabel;

  /// No description provided for @requestOtp.
  ///
  /// In fa, this message translates to:
  /// **'دریافت کد تأیید'**
  String get requestOtp;

  /// No description provided for @otpLabel.
  ///
  /// In fa, this message translates to:
  /// **'کد تأیید پیامک‌شده'**
  String get otpLabel;

  /// No description provided for @verifyAndLogin.
  ///
  /// In fa, this message translates to:
  /// **'تأیید و ورود'**
  String get verifyAndLogin;

  /// No description provided for @resendIn.
  ///
  /// In fa, this message translates to:
  /// **'ارسال مجدد کد تا {seconds} ثانیه دیگر'**
  String resendIn(String seconds);

  /// No description provided for @otpSentTo.
  ///
  /// In fa, this message translates to:
  /// **'کد تأیید به شماره {phone} پیامک شد.'**
  String otpSentTo(String phone);

  /// No description provided for @resendOtp.
  ///
  /// In fa, this message translates to:
  /// **'ارسال مجدد کد'**
  String get resendOtp;

  /// No description provided for @changePhone.
  ///
  /// In fa, this message translates to:
  /// **'تغییر شماره'**
  String get changePhone;

  /// No description provided for @invalidMobile.
  ///
  /// In fa, this message translates to:
  /// **'شماره موبایل معتبر نیست.'**
  String get invalidMobile;

  /// No description provided for @invalidOtp.
  ///
  /// In fa, this message translates to:
  /// **'کد تأیید را وارد کنید.'**
  String get invalidOtp;

  /// No description provided for @errorNetwork.
  ///
  /// In fa, this message translates to:
  /// **'خطا در اتصال به سرور؛ اتصال اینترنت را بررسی کنید.'**
  String get errorNetwork;

  /// No description provided for @errorTimeout.
  ///
  /// In fa, this message translates to:
  /// **'زمان انتظار پاسخ سرور به پایان رسید؛ دوباره تلاش کنید.'**
  String get errorTimeout;

  /// No description provided for @errorUnauthorized.
  ///
  /// In fa, this message translates to:
  /// **'احراز هویت لازم است.'**
  String get errorUnauthorized;

  /// No description provided for @errorForbidden.
  ///
  /// In fa, this message translates to:
  /// **'دسترسی غیرمجاز است.'**
  String get errorForbidden;

  /// No description provided for @errorNotFound.
  ///
  /// In fa, this message translates to:
  /// **'مورد درخواستی یافت نشد.'**
  String get errorNotFound;

  /// No description provided for @errorConflict.
  ///
  /// In fa, this message translates to:
  /// **'وضعیت ثبت با وضعیت فعلی هم‌خوانی ندارد.'**
  String get errorConflict;

  /// No description provided for @errorRateLimited.
  ///
  /// In fa, this message translates to:
  /// **'تعداد درخواست‌ها بیش از حد مجاز است؛ کمی صبر کنید.'**
  String get errorRateLimited;

  /// No description provided for @requiredField.
  ///
  /// In fa, this message translates to:
  /// **'این فیلد الزامی است.'**
  String get requiredField;

  /// No description provided for @invalidAmount.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ را به‌صورت عدد بزرگ‌تر از صفر وارد کنید.'**
  String get invalidAmount;

  /// No description provided for @errorValidation.
  ///
  /// In fa, this message translates to:
  /// **'اطلاعات ارسالی معتبر نیست.'**
  String get errorValidation;

  /// No description provided for @errorServer.
  ///
  /// In fa, this message translates to:
  /// **'خطای داخلی سرور؛ لطفاً دوباره تلاش کنید.'**
  String get errorServer;

  /// No description provided for @errorUnknown.
  ///
  /// In fa, this message translates to:
  /// **'خطایی رخ داد؛ دوباره تلاش کنید.'**
  String get errorUnknown;

  /// No description provided for @emptyStateTitle.
  ///
  /// In fa, this message translates to:
  /// **'هنوز موردی ثبت نشده است'**
  String get emptyStateTitle;

  /// No description provided for @emptyStateSubtitle.
  ///
  /// In fa, this message translates to:
  /// **'برای شروع، اولین مورد را اضافه کنید.'**
  String get emptyStateSubtitle;

  /// No description provided for @retry.
  ///
  /// In fa, this message translates to:
  /// **'تلاش دوباره'**
  String get retry;

  /// No description provided for @cancel.
  ///
  /// In fa, this message translates to:
  /// **'انصراف'**
  String get cancel;

  /// No description provided for @confirm.
  ///
  /// In fa, this message translates to:
  /// **'تأیید'**
  String get confirm;

  /// No description provided for @save.
  ///
  /// In fa, this message translates to:
  /// **'ذخیره'**
  String get save;

  /// No description provided for @edit.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش'**
  String get edit;

  /// No description provided for @delete.
  ///
  /// In fa, this message translates to:
  /// **'حذف'**
  String get delete;

  /// No description provided for @logout.
  ///
  /// In fa, this message translates to:
  /// **'خروج از حساب'**
  String get logout;

  /// No description provided for @selectDate.
  ///
  /// In fa, this message translates to:
  /// **'انتخاب تاریخ'**
  String get selectDate;

  /// No description provided for @tomanSuffix.
  ///
  /// In fa, this message translates to:
  /// **'تومان'**
  String get tomanSuffix;

  /// No description provided for @attachImage.
  ///
  /// In fa, this message translates to:
  /// **'افزودن تصویر'**
  String get attachImage;

  /// No description provided for @removeAttachment.
  ///
  /// In fa, this message translates to:
  /// **'حذف تصویر'**
  String get removeAttachment;

  /// No description provided for @notificationsTitle.
  ///
  /// In fa, this message translates to:
  /// **'اعلان‌ها'**
  String get notificationsTitle;

  /// No description provided for @markAllRead.
  ///
  /// In fa, this message translates to:
  /// **'علامت‌گذاری همه به‌عنوان خوانده‌شده'**
  String get markAllRead;

  /// No description provided for @unreadBadge.
  ///
  /// In fa, this message translates to:
  /// **'{count} ناخواندا'**
  String unreadBadge(int count);

  /// No description provided for @managerShellTitle.
  ///
  /// In fa, this message translates to:
  /// **'پنل مدیریت'**
  String get managerShellTitle;

  /// No description provided for @residentShellTitle.
  ///
  /// In fa, this message translates to:
  /// **'پنل ساکن'**
  String get residentShellTitle;

  /// No description provided for @shellUnderConstruction.
  ///
  /// In fa, this message translates to:
  /// **'این بخش در مراحل بعدی پیاده‌سازی می‌شود.'**
  String get shellUnderConstruction;

  /// No description provided for @buildingsTitle.
  ///
  /// In fa, this message translates to:
  /// **'ساختمان‌ها'**
  String get buildingsTitle;

  /// No description provided for @buildingName.
  ///
  /// In fa, this message translates to:
  /// **'نام ساختمان'**
  String get buildingName;

  /// No description provided for @buildingAddress.
  ///
  /// In fa, this message translates to:
  /// **'نشانی'**
  String get buildingAddress;

  /// No description provided for @blockCount.
  ///
  /// In fa, this message translates to:
  /// **'تعداد بلوک'**
  String get blockCount;

  /// No description provided for @floorCount.
  ///
  /// In fa, this message translates to:
  /// **'تعداد طبقات'**
  String get floorCount;

  /// No description provided for @unitCountLabel.
  ///
  /// In fa, this message translates to:
  /// **'تعداد واحدها'**
  String get unitCountLabel;

  /// No description provided for @builtYear.
  ///
  /// In fa, this message translates to:
  /// **'سال ساخت'**
  String get builtYear;

  /// No description provided for @managerPhoneLabel.
  ///
  /// In fa, this message translates to:
  /// **'شماره مدیر'**
  String get managerPhoneLabel;

  /// No description provided for @emergencyPhoneLabel.
  ///
  /// In fa, this message translates to:
  /// **'شماره اضطراری'**
  String get emergencyPhoneLabel;

  /// No description provided for @notesLabel.
  ///
  /// In fa, this message translates to:
  /// **'یادداشت'**
  String get notesLabel;

  /// No description provided for @addBuilding.
  ///
  /// In fa, this message translates to:
  /// **'افزودن ساختمان'**
  String get addBuilding;

  /// No description provided for @editBuilding.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش ساختمان'**
  String get editBuilding;

  /// No description provided for @archiveBuilding.
  ///
  /// In fa, this message translates to:
  /// **'آرشیو ساختمان'**
  String get archiveBuilding;

  /// No description provided for @archiveConfirm.
  ///
  /// In fa, this message translates to:
  /// **'آیا از آرشیو این مورد مطمئن هستید؟'**
  String get archiveConfirm;

  /// No description provided for @unitsTitle.
  ///
  /// In fa, this message translates to:
  /// **'واحدها'**
  String get unitsTitle;

  /// No description provided for @unitNumber.
  ///
  /// In fa, this message translates to:
  /// **'شماره واحد'**
  String get unitNumber;

  /// No description provided for @unitBlock.
  ///
  /// In fa, this message translates to:
  /// **'بلوک'**
  String get unitBlock;

  /// No description provided for @unitFloor.
  ///
  /// In fa, this message translates to:
  /// **'طبقه'**
  String get unitFloor;

  /// No description provided for @areaM2.
  ///
  /// In fa, this message translates to:
  /// **'متراژ (متر مربع)'**
  String get areaM2;

  /// No description provided for @parkingCount.
  ///
  /// In fa, this message translates to:
  /// **'تعداد پارکینگ'**
  String get parkingCount;

  /// No description provided for @parkingNumbers.
  ///
  /// In fa, this message translates to:
  /// **'شماره پارکینگ‌ها'**
  String get parkingNumbers;

  /// No description provided for @storageCount.
  ///
  /// In fa, this message translates to:
  /// **'تعداد انباری'**
  String get storageCount;

  /// No description provided for @storageNumbers.
  ///
  /// In fa, this message translates to:
  /// **'شماره انباری‌ها'**
  String get storageNumbers;

  /// No description provided for @unitStatus.
  ///
  /// In fa, this message translates to:
  /// **'وضعیت واحد'**
  String get unitStatus;

  /// No description provided for @statusActive.
  ///
  /// In fa, this message translates to:
  /// **'فعال'**
  String get statusActive;

  /// No description provided for @statusVacant.
  ///
  /// In fa, this message translates to:
  /// **'خالی'**
  String get statusVacant;

  /// No description provided for @statusOccupied.
  ///
  /// In fa, this message translates to:
  /// **'مسکونی'**
  String get statusOccupied;

  /// No description provided for @statusInactive.
  ///
  /// In fa, this message translates to:
  /// **'غیرفعال'**
  String get statusInactive;

  /// No description provided for @addUnit.
  ///
  /// In fa, this message translates to:
  /// **'افزودن واحد'**
  String get addUnit;

  /// No description provided for @editUnit.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش واحد'**
  String get editUnit;

  /// No description provided for @searchUnits.
  ///
  /// In fa, this message translates to:
  /// **'جست‌وجوی واحد…'**
  String get searchUnits;

  /// No description provided for @allBlocks.
  ///
  /// In fa, this message translates to:
  /// **'همه بلوک‌ها'**
  String get allBlocks;

  /// No description provided for @allStatuses.
  ///
  /// In fa, this message translates to:
  /// **'همه وضعیت‌ها'**
  String get allStatuses;

  /// No description provided for @changeHistory.
  ///
  /// In fa, this message translates to:
  /// **'تاریخچه تغییرات'**
  String get changeHistory;

  /// No description provided for @noChangeHistory.
  ///
  /// In fa, this message translates to:
  /// **'تغییری ثبت نشده است.'**
  String get noChangeHistory;

  /// No description provided for @invalidFloor.
  ///
  /// In fa, this message translates to:
  /// **'طبقه را صحیح وارد کنید.'**
  String get invalidFloor;

  /// No description provided for @invalidNumber.
  ///
  /// In fa, this message translates to:
  /// **'عدد صحیح وارد کنید.'**
  String get invalidNumber;

  /// No description provided for @peopleTitle.
  ///
  /// In fa, this message translates to:
  /// **'افراد و ساکنان'**
  String get peopleTitle;

  /// No description provided for @addPerson.
  ///
  /// In fa, this message translates to:
  /// **'افزودن شخص'**
  String get addPerson;

  /// No description provided for @editPerson.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش شخص'**
  String get editPerson;

  /// No description provided for @personName.
  ///
  /// In fa, this message translates to:
  /// **'نام و نام خانوادگی'**
  String get personName;

  /// No description provided for @personPhone.
  ///
  /// In fa, this message translates to:
  /// **'شماره موبایل'**
  String get personPhone;

  /// No description provided for @personNationalId.
  ///
  /// In fa, this message translates to:
  /// **'کد ملی'**
  String get personNationalId;

  /// No description provided for @invalidNationalId.
  ///
  /// In fa, this message translates to:
  /// **'کد ملی باید ۱۰ رقم باشد.'**
  String get invalidNationalId;

  /// No description provided for @relationshipLabel.
  ///
  /// In fa, this message translates to:
  /// **'نسبت ساکن'**
  String get relationshipLabel;

  /// No description provided for @relOwner.
  ///
  /// In fa, this message translates to:
  /// **'مالک'**
  String get relOwner;

  /// No description provided for @relTenant.
  ///
  /// In fa, this message translates to:
  /// **'مستأجر'**
  String get relTenant;

  /// No description provided for @relNonResidentOwner.
  ///
  /// In fa, this message translates to:
  /// **'مالک غیرمقیم'**
  String get relNonResidentOwner;

  /// No description provided for @selectPerson.
  ///
  /// In fa, this message translates to:
  /// **'انتخاب شخص'**
  String get selectPerson;

  /// No description provided for @addOccupancy.
  ///
  /// In fa, this message translates to:
  /// **'ثبت سکونت'**
  String get addOccupancy;

  /// No description provided for @occupancyTitle.
  ///
  /// In fa, this message translates to:
  /// **'ساکنان و سابقه سکونت'**
  String get occupancyTitle;

  /// No description provided for @occupancyStart.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ شروع سکونت'**
  String get occupancyStart;

  /// No description provided for @occupancyActive.
  ///
  /// In fa, this message translates to:
  /// **'فعال'**
  String get occupancyActive;

  /// No description provided for @occupancyEnded.
  ///
  /// In fa, this message translates to:
  /// **'پایان‌یافته'**
  String get occupancyEnded;

  /// No description provided for @endOccupancy.
  ///
  /// In fa, this message translates to:
  /// **'پایان سکونت'**
  String get endOccupancy;

  /// No description provided for @occupantCountTitle.
  ///
  /// In fa, this message translates to:
  /// **'تعداد ساکن'**
  String get occupantCountTitle;

  /// No description provided for @recordOccupantCount.
  ///
  /// In fa, this message translates to:
  /// **'ثبت تعداد جدید'**
  String get recordOccupantCount;

  /// No description provided for @effectiveFrom.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ اثر'**
  String get effectiveFrom;

  /// No description provided for @occupancyNeedsPerson.
  ///
  /// In fa, this message translates to:
  /// **'هنوز شخصی برای این ساختمان ثبت نشده است؛ ابتدا یک شخص اضافه کنید.'**
  String get occupancyNeedsPerson;

  /// No description provided for @billingTitle.
  ///
  /// In fa, this message translates to:
  /// **'شارژها'**
  String get billingTitle;

  /// No description provided for @addPeriod.
  ///
  /// In fa, this message translates to:
  /// **'دوره جدید'**
  String get addPeriod;

  /// No description provided for @editPeriod.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش دوره'**
  String get editPeriod;

  /// No description provided for @periodTitleLabel.
  ///
  /// In fa, this message translates to:
  /// **'عنوان دوره'**
  String get periodTitleLabel;

  /// No description provided for @periodStart.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ شروع'**
  String get periodStart;

  /// No description provided for @periodEnd.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ پایان'**
  String get periodEnd;

  /// No description provided for @periodDue.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ سررسید'**
  String get periodDue;

  /// No description provided for @lateFeeType.
  ///
  /// In fa, this message translates to:
  /// **'نوع دیرکرد'**
  String get lateFeeType;

  /// No description provided for @lateFeeNone.
  ///
  /// In fa, this message translates to:
  /// **'بدون دیرکرد'**
  String get lateFeeNone;

  /// No description provided for @lateFeeFixed.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ ثابت'**
  String get lateFeeFixed;

  /// No description provided for @lateFeePercent.
  ///
  /// In fa, this message translates to:
  /// **'درصدی'**
  String get lateFeePercent;

  /// No description provided for @lateFeePerDay.
  ///
  /// In fa, this message translates to:
  /// **'روزانه'**
  String get lateFeePerDay;

  /// No description provided for @lateFeeValue.
  ///
  /// In fa, this message translates to:
  /// **'مقدار دیرکرد'**
  String get lateFeeValue;

  /// No description provided for @costItemsTitle.
  ///
  /// In fa, this message translates to:
  /// **'اقلام هزینه'**
  String get costItemsTitle;

  /// No description provided for @addCostItem.
  ///
  /// In fa, this message translates to:
  /// **'افزودن قلم هزینه'**
  String get addCostItem;

  /// No description provided for @editCostItem.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش قلم هزینه'**
  String get editCostItem;

  /// No description provided for @costItemTitle.
  ///
  /// In fa, this message translates to:
  /// **'عنوان قلم'**
  String get costItemTitle;

  /// No description provided for @costItemAmount.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ'**
  String get costItemAmount;

  /// No description provided for @calcMethod.
  ///
  /// In fa, this message translates to:
  /// **'روش محاسبه'**
  String get calcMethod;

  /// No description provided for @methodEqual.
  ///
  /// In fa, this message translates to:
  /// **'مساوی'**
  String get methodEqual;

  /// No description provided for @methodPerOccupant.
  ///
  /// In fa, this message translates to:
  /// **'سرشمار (ساکن)'**
  String get methodPerOccupant;

  /// No description provided for @methodPerArea.
  ///
  /// In fa, this message translates to:
  /// **'متراژ'**
  String get methodPerArea;

  /// No description provided for @methodFixed.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ ثابت'**
  String get methodFixed;

  /// No description provided for @methodSpecificUnits.
  ///
  /// In fa, this message translates to:
  /// **'واحدهای مشخص'**
  String get methodSpecificUnits;

  /// No description provided for @methodCombined.
  ///
  /// In fa, this message translates to:
  /// **'ترکیبی'**
  String get methodCombined;

  /// No description provided for @fixedPerUnit.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ هر واحد'**
  String get fixedPerUnit;

  /// No description provided for @includeVacant.
  ///
  /// In fa, this message translates to:
  /// **'شامل واحدهای خالی'**
  String get includeVacant;

  /// No description provided for @selectUnits.
  ///
  /// In fa, this message translates to:
  /// **'انتخاب واحدها'**
  String get selectUnits;

  /// No description provided for @comboWeights.
  ///
  /// In fa, this message translates to:
  /// **'ترکیب وزن‌ها'**
  String get comboWeights;

  /// No description provided for @addWeight.
  ///
  /// In fa, this message translates to:
  /// **'افزودن سهم'**
  String get addWeight;

  /// No description provided for @comboWeightsInvalid.
  ///
  /// In fa, this message translates to:
  /// **'جمع وزن‌ها باید ۱۰۰ باشد'**
  String get comboWeightsInvalid;

  /// No description provided for @calculate.
  ///
  /// In fa, this message translates to:
  /// **'محاسبه'**
  String get calculate;

  /// No description provided for @recalculate.
  ///
  /// In fa, this message translates to:
  /// **'محاسبه مجدد'**
  String get recalculate;

  /// No description provided for @previewTitle.
  ///
  /// In fa, this message translates to:
  /// **'پیش‌نمایش محاسبه'**
  String get previewTitle;

  /// No description provided for @reconciliationOk.
  ///
  /// In fa, this message translates to:
  /// **'مغایرت ندارد'**
  String get reconciliationOk;

  /// No description provided for @reconciliationBad.
  ///
  /// In fa, this message translates to:
  /// **'مغایرت در جمع اقلام!'**
  String get reconciliationBad;

  /// No description provided for @exactShare.
  ///
  /// In fa, this message translates to:
  /// **'سهم دقیق'**
  String get exactShare;

  /// No description provided for @issue.
  ///
  /// In fa, this message translates to:
  /// **'صدور صورتحساب‌ها'**
  String get issue;

  /// No description provided for @issueConfirm.
  ///
  /// In fa, this message translates to:
  /// **'پس از صدور، صورتحساب‌ها تغییرناپذیرند و ساکنان مطلع می‌شوند. ادامه می‌دهید؟'**
  String get issueConfirm;

  /// No description provided for @issuedOk.
  ///
  /// In fa, this message translates to:
  /// **'صورتحساب‌ها صادر شد'**
  String get issuedOk;

  /// No description provided for @reopen.
  ///
  /// In fa, this message translates to:
  /// **'بازگشایی دوره'**
  String get reopen;

  /// No description provided for @reopenConfirm.
  ///
  /// In fa, this message translates to:
  /// **'پیش‌نمایش محاسبه حذف و دوره به پیش‌نویس بازمی‌گردد. ادامه می‌دهید؟'**
  String get reopenConfirm;

  /// No description provided for @closePeriod.
  ///
  /// In fa, this message translates to:
  /// **'بستن دوره'**
  String get closePeriod;

  /// No description provided for @closeConfirm.
  ///
  /// In fa, this message translates to:
  /// **'با بستن دوره دیگر هیچ تغییری امکان‌پذیر نیست. ادامه می‌دهید؟'**
  String get closeConfirm;

  /// No description provided for @invoicesTitle.
  ///
  /// In fa, this message translates to:
  /// **'صورتحساب‌ها'**
  String get invoicesTitle;

  /// No description provided for @baseAmount.
  ///
  /// In fa, this message translates to:
  /// **'شارژ دوره'**
  String get baseAmount;

  /// No description provided for @priorDebt.
  ///
  /// In fa, this message translates to:
  /// **'بدهی قبلی'**
  String get priorDebt;

  /// No description provided for @lateFeeAmount.
  ///
  /// In fa, this message translates to:
  /// **'دیرکرد'**
  String get lateFeeAmount;

  /// No description provided for @creditAmount.
  ///
  /// In fa, this message translates to:
  /// **'اعتبار'**
  String get creditAmount;

  /// No description provided for @finalAmount.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ نهایی'**
  String get finalAmount;

  /// No description provided for @paidAmount.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت‌شده'**
  String get paidAmount;

  /// No description provided for @issueDateLabel.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ صدور'**
  String get issueDateLabel;

  /// No description provided for @dueDateLabel.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ سررسید'**
  String get dueDateLabel;

  /// No description provided for @itemKindAdjustment.
  ///
  /// In fa, this message translates to:
  /// **'اصلاحیه'**
  String get itemKindAdjustment;

  /// No description provided for @adjustmentDebit.
  ///
  /// In fa, this message translates to:
  /// **'بدهکار'**
  String get adjustmentDebit;

  /// No description provided for @adjustmentCredit.
  ///
  /// In fa, this message translates to:
  /// **'بستانکار'**
  String get adjustmentCredit;

  /// No description provided for @addAdjustment.
  ///
  /// In fa, this message translates to:
  /// **'ثبت اصلاحیه'**
  String get addAdjustment;

  /// No description provided for @adjustmentReason.
  ///
  /// In fa, this message translates to:
  /// **'دلیل اصلاحیه'**
  String get adjustmentReason;

  /// No description provided for @cancelInvoice.
  ///
  /// In fa, this message translates to:
  /// **'ابطال صورتحساب'**
  String get cancelInvoice;

  /// No description provided for @cancelInvoiceConfirm.
  ///
  /// In fa, this message translates to:
  /// **'صورتحساب ابطال شود؟ ردیف و سابقه آن حفظ خواهد شد.'**
  String get cancelInvoiceConfirm;

  /// No description provided for @cancelReason.
  ///
  /// In fa, this message translates to:
  /// **'دلیل ابطال'**
  String get cancelReason;

  /// No description provided for @myCharges.
  ///
  /// In fa, this message translates to:
  /// **'شارژهای من'**
  String get myCharges;

  /// No description provided for @paymentsTitle.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت‌ها'**
  String get paymentsTitle;

  /// No description provided for @recordPayment.
  ///
  /// In fa, this message translates to:
  /// **'ثبت پرداخت'**
  String get recordPayment;

  /// No description provided for @paymentAmountLabel.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ پرداخت (تومان)'**
  String get paymentAmountLabel;

  /// No description provided for @paymentDateLabel.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ پرداخت'**
  String get paymentDateLabel;

  /// No description provided for @trackingNumberLabel.
  ///
  /// In fa, this message translates to:
  /// **'شماره پیگیری / رسید (اختیاری)'**
  String get trackingNumberLabel;

  /// No description provided for @paymentRecordedOk.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت با موفقیت ثبت شد'**
  String get paymentRecordedOk;

  /// No description provided for @ledgerTitle.
  ///
  /// In fa, this message translates to:
  /// **'دفتر پرداخت‌ها'**
  String get ledgerTitle;

  /// No description provided for @filterMethodLabel.
  ///
  /// In fa, this message translates to:
  /// **'روش پرداخت'**
  String get filterMethodLabel;

  /// No description provided for @filterAll.
  ///
  /// In fa, this message translates to:
  /// **'همه'**
  String get filterAll;

  /// No description provided for @filterFromDate.
  ///
  /// In fa, this message translates to:
  /// **'از تاریخ'**
  String get filterFromDate;

  /// No description provided for @filterToDate.
  ///
  /// In fa, this message translates to:
  /// **'تا تاریخ'**
  String get filterToDate;

  /// No description provided for @payTitle.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت آنلاین'**
  String get payTitle;

  /// No description provided for @outstandingLabel.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ باقی‌مانده'**
  String get outstandingLabel;

  /// No description provided for @payLaunchConfirm.
  ///
  /// In fa, this message translates to:
  /// **'به درگاه پرداخت منتقل می‌شوید. پس از بازگشت، وضعیت پرداخت بررسی می‌شود.'**
  String get payLaunchConfirm;

  /// No description provided for @paySuccess.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت با موفقیت انجام شد'**
  String get paySuccess;

  /// No description provided for @payFailed.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت ناموفق بود یا تأیید نشد'**
  String get payFailed;

  /// No description provided for @launchFailed.
  ///
  /// In fa, this message translates to:
  /// **'باز کردن درگاه پرداخت ممکن نشد'**
  String get launchFailed;

  /// No description provided for @receiptTitle.
  ///
  /// In fa, this message translates to:
  /// **'رسید پرداخت'**
  String get receiptTitle;

  /// No description provided for @paymentHistoryTitle.
  ///
  /// In fa, this message translates to:
  /// **'تاریخچه پرداخت‌ها'**
  String get paymentHistoryTitle;

  /// No description provided for @unitLabel.
  ///
  /// In fa, this message translates to:
  /// **'واحد'**
  String get unitLabel;

  /// No description provided for @payNow.
  ///
  /// In fa, this message translates to:
  /// **'پرداخت آنلاین'**
  String get payNow;

  /// No description provided for @noPayments.
  ///
  /// In fa, this message translates to:
  /// **'هنوز پرداختی ثبت نشده است'**
  String get noPayments;

  /// No description provided for @expensesTitle.
  ///
  /// In fa, this message translates to:
  /// **'هزینه‌ها'**
  String get expensesTitle;

  /// No description provided for @financialReport.
  ///
  /// In fa, this message translates to:
  /// **'گزارش مالی'**
  String get financialReport;

  /// No description provided for @addExpense.
  ///
  /// In fa, this message translates to:
  /// **'ثبت هزینه'**
  String get addExpense;

  /// No description provided for @editExpense.
  ///
  /// In fa, this message translates to:
  /// **'ویرایش هزینه'**
  String get editExpense;

  /// No description provided for @expenseTitleLabel.
  ///
  /// In fa, this message translates to:
  /// **'عنوان هزینه'**
  String get expenseTitleLabel;

  /// No description provided for @expenseCategoryLabel.
  ///
  /// In fa, this message translates to:
  /// **'دسته‌بندی'**
  String get expenseCategoryLabel;

  /// No description provided for @expenseAmountLabel.
  ///
  /// In fa, this message translates to:
  /// **'مبلغ (تومان)'**
  String get expenseAmountLabel;

  /// No description provided for @expenseDateLabel.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ هزینه'**
  String get expenseDateLabel;

  /// No description provided for @expenseApprovalLabel.
  ///
  /// In fa, this message translates to:
  /// **'وضعیت تأیید'**
  String get expenseApprovalLabel;

  /// No description provided for @expenseDescriptionLabel.
  ///
  /// In fa, this message translates to:
  /// **'توضیحات (اختیاری)'**
  String get expenseDescriptionLabel;

  /// No description provided for @filterCategoryLabel.
  ///
  /// In fa, this message translates to:
  /// **'دسته‌بندی'**
  String get filterCategoryLabel;

  /// No description provided for @filterApprovalLabel.
  ///
  /// In fa, this message translates to:
  /// **'وضعیت تأیید'**
  String get filterApprovalLabel;

  /// No description provided for @noExpenses.
  ///
  /// In fa, this message translates to:
  /// **'هنوز هزینه‌ای ثبت نشده است'**
  String get noExpenses;

  /// No description provided for @expenseSavedOk.
  ///
  /// In fa, this message translates to:
  /// **'هزینه با موفقیت ثبت شد'**
  String get expenseSavedOk;

  /// No description provided for @expenseDeletedOk.
  ///
  /// In fa, this message translates to:
  /// **'هزینه حذف شد'**
  String get expenseDeletedOk;

  /// No description provided for @deleteExpense.
  ///
  /// In fa, this message translates to:
  /// **'حذف هزینه'**
  String get deleteExpense;

  /// No description provided for @deleteExpenseConfirm.
  ///
  /// In fa, this message translates to:
  /// **'این هزینه حذف شود؟ ردیف آن از گزارش‌ها حذف می‌شود ولی سابقه مالی حفظ می‌شود.'**
  String get deleteExpenseConfirm;

  /// No description provided for @reportMonthLabel.
  ///
  /// In fa, this message translates to:
  /// **'انتخاب ماه'**
  String get reportMonthLabel;

  /// No description provided for @reportMonthlyIncome.
  ///
  /// In fa, this message translates to:
  /// **'درآمد این ماه'**
  String get reportMonthlyIncome;

  /// No description provided for @reportMonthlyExpense.
  ///
  /// In fa, this message translates to:
  /// **'هزینه این ماه'**
  String get reportMonthlyExpense;

  /// No description provided for @reportNet.
  ///
  /// In fa, this message translates to:
  /// **'تراز این ماه'**
  String get reportNet;

  /// No description provided for @reportTotalDebt.
  ///
  /// In fa, this message translates to:
  /// **'مجموع بدهی ساکنان'**
  String get reportTotalDebt;

  /// No description provided for @reportTotalPayments.
  ///
  /// In fa, this message translates to:
  /// **'مجموع پرداخت‌ها'**
  String get reportTotalPayments;

  /// No description provided for @reportTotalExpenses.
  ///
  /// In fa, this message translates to:
  /// **'مجموع هزینه‌ها'**
  String get reportTotalExpenses;

  /// No description provided for @devTestCode.
  ///
  /// In fa, this message translates to:
  /// **'کد آزمایشی (حالت توسعه): {code}'**
  String devTestCode(String code);

  /// No description provided for @maintenanceTitle.
  ///
  /// In fa, this message translates to:
  /// **'درخواست‌های نگهداری'**
  String get maintenanceTitle;

  /// No description provided for @myMaintenanceTitle.
  ///
  /// In fa, this message translates to:
  /// **'درخواست‌های من'**
  String get myMaintenanceTitle;

  /// No description provided for @addMaintenanceRequest.
  ///
  /// In fa, this message translates to:
  /// **'ثبت درخواست'**
  String get addMaintenanceRequest;

  /// No description provided for @newMaintenanceTitle.
  ///
  /// In fa, this message translates to:
  /// **'ثبت درخواست جدید'**
  String get newMaintenanceTitle;

  /// No description provided for @maintenanceDetailTitle.
  ///
  /// In fa, this message translates to:
  /// **'جزئیات درخواست'**
  String get maintenanceDetailTitle;

  /// No description provided for @maintenanceTitleLabel.
  ///
  /// In fa, this message translates to:
  /// **'عنوان درخواست'**
  String get maintenanceTitleLabel;

  /// No description provided for @maintenanceTitleHint.
  ///
  /// In fa, this message translates to:
  /// **'مثلاً: آسانسور لرزش دارد'**
  String get maintenanceTitleHint;

  /// No description provided for @maintenanceTitleRequired.
  ///
  /// In fa, this message translates to:
  /// **'عنوان الزامی است'**
  String get maintenanceTitleRequired;

  /// No description provided for @maintenanceCategoryLabel.
  ///
  /// In fa, this message translates to:
  /// **'دسته‌بندی'**
  String get maintenanceCategoryLabel;

  /// No description provided for @maintenancePriorityLabel.
  ///
  /// In fa, this message translates to:
  /// **'اولویت'**
  String get maintenancePriorityLabel;

  /// No description provided for @maintenanceLocationLabel.
  ///
  /// In fa, this message translates to:
  /// **'محل (اختیاری)'**
  String get maintenanceLocationLabel;

  /// No description provided for @maintenanceLocationHint.
  ///
  /// In fa, this message translates to:
  /// **'مثلاً: طبقه ۳، آسانسور'**
  String get maintenanceLocationHint;

  /// No description provided for @maintenanceDescriptionLabel.
  ///
  /// In fa, this message translates to:
  /// **'توضیحات (اختیاری)'**
  String get maintenanceDescriptionLabel;

  /// No description provided for @maintenancePhotoLabel.
  ///
  /// In fa, this message translates to:
  /// **'تصویر (اختیاری)'**
  String get maintenancePhotoLabel;

  /// No description provided for @maintenanceSubmitSuccess.
  ///
  /// In fa, this message translates to:
  /// **'درخواست با موفقیت ثبت شد'**
  String get maintenanceSubmitSuccess;

  /// No description provided for @maintenanceSubmitError.
  ///
  /// In fa, this message translates to:
  /// **'خطا در ثبت درخواست'**
  String get maintenanceSubmitError;

  /// No description provided for @maintenanceLoadError.
  ///
  /// In fa, this message translates to:
  /// **'خطا در دریافت'**
  String get maintenanceLoadError;

  /// No description provided for @maintenanceEmptyTitle.
  ///
  /// In fa, this message translates to:
  /// **'هنوز درخواستی ثبت نکرده‌اید'**
  String get maintenanceEmptyTitle;

  /// No description provided for @maintenanceEmptySubtitle.
  ///
  /// In fa, this message translates to:
  /// **'برای ثبت مشکل جدید، دکمه «ثبت درخواست» را بزنید.'**
  String get maintenanceEmptySubtitle;

  /// No description provided for @maintenanceNoRequests.
  ///
  /// In fa, this message translates to:
  /// **'درخواستی یافت نشد'**
  String get maintenanceNoRequests;

  /// No description provided for @maintenanceNoRequestsSubtitle.
  ///
  /// In fa, this message translates to:
  /// **'با فیلترهای فعلی هیچ درخواستی وجود ندارد.'**
  String get maintenanceNoRequestsSubtitle;

  /// No description provided for @maintenanceFilterAllStatuses.
  ///
  /// In fa, this message translates to:
  /// **'همه وضعیت‌ها'**
  String get maintenanceFilterAllStatuses;

  /// No description provided for @maintenanceFilterAllPriorities.
  ///
  /// In fa, this message translates to:
  /// **'همه اولویت‌ها'**
  String get maintenanceFilterAllPriorities;

  /// No description provided for @maintenanceFilterAllCategories.
  ///
  /// In fa, this message translates to:
  /// **'همه دسته‌ها'**
  String get maintenanceFilterAllCategories;

  /// No description provided for @maintenanceCreatedAt.
  ///
  /// In fa, this message translates to:
  /// **'تاریخ ثبت: {date}'**
  String maintenanceCreatedAt(String date);

  /// No description provided for @maintenanceLocation.
  ///
  /// In fa, this message translates to:
  /// **'محل: {location}'**
  String maintenanceLocation(String location);

  /// No description provided for @maintenanceRecordedCost.
  ///
  /// In fa, this message translates to:
  /// **'هزینه ثبت‌شده: {amount} تومان'**
  String maintenanceRecordedCost(String amount);

  /// No description provided for @maintenanceChangeStatus.
  ///
  /// In fa, this message translates to:
  /// **'تغییر وضعیت'**
  String get maintenanceChangeStatus;

  /// No description provided for @maintenanceChangeStatusConfirm.
  ///
  /// In fa, this message translates to:
  /// **'وضعیت درخواست به «{status}» تغییر کند؟'**
  String maintenanceChangeStatusConfirm(String status);

  /// No description provided for @maintenanceStatusChangedTo.
  ///
  /// In fa, this message translates to:
  /// **'وضعیت به «{status}» تغییر کرد'**
  String maintenanceStatusChangedTo(String status);

  /// No description provided for @maintenanceStatusChangeNotAllowed.
  ///
  /// In fa, this message translates to:
  /// **'تغییر وضعیت مجاز نیست'**
  String get maintenanceStatusChangeNotAllowed;

  /// No description provided for @maintenanceMetaTitle.
  ///
  /// In fa, this message translates to:
  /// **'مسئول و هزینه'**
  String get maintenanceMetaTitle;

  /// No description provided for @maintenanceAssigneeLabel.
  ///
  /// In fa, this message translates to:
  /// **'شناسه مسئول (UUID)'**
  String get maintenanceAssigneeLabel;

  /// No description provided for @maintenanceAssigneeHint.
  ///
  /// In fa, this message translates to:
  /// **'اختیاری'**
  String get maintenanceAssigneeHint;

  /// No description provided for @maintenanceCostLabel.
  ///
  /// In fa, this message translates to:
  /// **'هزینه ثبت‌شده (تومان)'**
  String get maintenanceCostLabel;

  /// No description provided for @maintenanceCostHint.
  ///
  /// In fa, this message translates to:
  /// **'مثلاً ۵۰۰۰۰۰'**
  String get maintenanceCostHint;

  /// No description provided for @maintenanceNotesLabel.
  ///
  /// In fa, this message translates to:
  /// **'یادداشت مدیر'**
  String get maintenanceNotesLabel;

  /// No description provided for @maintenanceSaveMeta.
  ///
  /// In fa, this message translates to:
  /// **'ذخیره مسئول / هزینه / یادداشت'**
  String get maintenanceSaveMeta;

  /// No description provided for @maintenanceSaved.
  ///
  /// In fa, this message translates to:
  /// **'تغییرات ذخیره شد'**
  String get maintenanceSaved;

  /// No description provided for @maintenanceSaveError.
  ///
  /// In fa, this message translates to:
  /// **'خطا در ذخیره'**
  String get maintenanceSaveError;

  /// No description provided for @maintenanceBack.
  ///
  /// In fa, this message translates to:
  /// **'بازگشت'**
  String get maintenanceBack;

  /// No description provided for @maintenanceClosedChip.
  ///
  /// In fa, this message translates to:
  /// **'این درخواست بسته شده است'**
  String get maintenanceClosedChip;

  /// No description provided for @maintenanceStatusNew.
  ///
  /// In fa, this message translates to:
  /// **'ثبت‌شده'**
  String get maintenanceStatusNew;

  /// No description provided for @maintenanceStatusUnderReview.
  ///
  /// In fa, this message translates to:
  /// **'در حال بررسی'**
  String get maintenanceStatusUnderReview;

  /// No description provided for @maintenanceStatusInProgress.
  ///
  /// In fa, this message translates to:
  /// **'در حال انجام'**
  String get maintenanceStatusInProgress;

  /// No description provided for @maintenanceStatusDone.
  ///
  /// In fa, this message translates to:
  /// **'انجام‌شده'**
  String get maintenanceStatusDone;

  /// No description provided for @maintenanceStatusClosed.
  ///
  /// In fa, this message translates to:
  /// **'بسته‌شده'**
  String get maintenanceStatusClosed;

  /// No description provided for @maintenanceCategoryElevator.
  ///
  /// In fa, this message translates to:
  /// **'آسانسور'**
  String get maintenanceCategoryElevator;

  /// No description provided for @maintenanceCategoryUtilities.
  ///
  /// In fa, this message translates to:
  /// **'تأسیسات'**
  String get maintenanceCategoryUtilities;

  /// No description provided for @maintenanceCategoryElectrical.
  ///
  /// In fa, this message translates to:
  /// **'برق'**
  String get maintenanceCategoryElectrical;

  /// No description provided for @maintenanceCategoryWater.
  ///
  /// In fa, this message translates to:
  /// **'آب'**
  String get maintenanceCategoryWater;

  /// No description provided for @maintenanceCategoryCleaning.
  ///
  /// In fa, this message translates to:
  /// **'نظافت'**
  String get maintenanceCategoryCleaning;

  /// No description provided for @maintenanceCategoryCommonArea.
  ///
  /// In fa, this message translates to:
  /// **'مشاعات'**
  String get maintenanceCategoryCommonArea;

  /// No description provided for @maintenanceCategoryParking.
  ///
  /// In fa, this message translates to:
  /// **'پارکینگ'**
  String get maintenanceCategoryParking;

  /// No description provided for @maintenanceCategoryOther.
  ///
  /// In fa, this message translates to:
  /// **'سایر'**
  String get maintenanceCategoryOther;

  /// No description provided for @maintenancePriorityNormal.
  ///
  /// In fa, this message translates to:
  /// **'عادی'**
  String get maintenancePriorityNormal;

  /// No description provided for @maintenancePriorityImportant.
  ///
  /// In fa, this message translates to:
  /// **'مهم'**
  String get maintenancePriorityImportant;

  /// No description provided for @maintenancePriorityUrgent.
  ///
  /// In fa, this message translates to:
  /// **'فوری'**
  String get maintenancePriorityUrgent;

  /// No description provided for @maintenanceResidentMenu.
  ///
  /// In fa, this message translates to:
  /// **'درخواست‌های نگهداری'**
  String get maintenanceResidentMenu;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['fa'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'fa':
      return AppLocalizationsFa();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
