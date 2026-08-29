/// US3 domain models (contracts/api.md "People & Occupancy P0-02").
class Person {
  Person({
    required this.id,
    required this.buildingId,
    required this.fullName,
    this.phone,
    this.nationalId,
  });

  factory Person.fromJson(Map<String, dynamic> j) => Person(
    id: j['id'] as String,
    buildingId: (j['building_id'] ?? '') as String,
    fullName: (j['full_name'] ?? '') as String,
    phone: j['phone'] as String?,
    nationalId: j['national_id'] as String?,
  );

  final String id;
  final String buildingId;
  final String fullName;
  final String? phone;
  final String? nationalId;
}

/// Occupancy relationship tokens stay locale-neutral on the wire; Persian
/// labels come from [relationshipLabel] in the screens.
const occupancyRelationships = ['owner', 'tenant', 'non_resident_owner'];

class Occupancy {
  Occupancy({
    required this.id,
    required this.unitId,
    required this.personId,
    required this.relationship,
    required this.startDate,
    this.endDate,
    required this.isActive,
    this.person,
  });

  factory Occupancy.fromJson(Map<String, dynamic> j) => Occupancy(
    id: j['id'] as String,
    unitId: (j['unit_id'] ?? '') as String,
    personId: (j['person_id'] ?? '') as String,
    relationship: (j['relationship'] ?? '') as String,
    startDate: (j['start_date'] ?? '') as String,
    endDate: j['end_date'] as String?,
    isActive: (j['is_active'] ?? false) as bool,
    person: j['person'] == null
        ? null
        : Person.fromJson(Map<String, dynamic>.from(j['person'] as Map)),
  );

  final String id;
  final String unitId;
  final String personId;
  final String relationship;
  final String startDate;
  final String? endDate;
  final bool isActive;
  final Person? person;
}

class OccupantCountEntry {
  OccupantCountEntry({
    required this.id,
    required this.unitId,
    required this.count,
    required this.effectiveFrom,
  });

  factory OccupantCountEntry.fromJson(Map<String, dynamic> j) =>
      OccupantCountEntry(
        id: j['id'] as String,
        unitId: (j['unit_id'] ?? '') as String,
        count: (j['occupant_count'] ?? 0) as int,
        effectiveFrom: (j['effective_from'] ?? '') as String,
      );

  final String id;
  final String unitId;
  final int count;
  final String effectiveFrom;
}
