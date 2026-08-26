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
