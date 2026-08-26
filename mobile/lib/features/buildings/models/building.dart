/// US2 domain models (contracts/api.md "Buildings & Units P0-01").
class Building {
  Building({
    required this.id,
    required this.name,
    this.address,
    this.blockCount = 0,
    this.floorCount = 0,
    this.unitCount = 0,
    this.builtYear,
    this.managerPhone,
    this.emergencyPhone,
    this.notes,
  });

  factory Building.fromJson(Map<String, dynamic> j) => Building(
    id: j['id'] as String,
    name: (j['name'] ?? '') as String,
    address: j['address'] as String?,
    blockCount: (j['block_count'] ?? 0) as int,
    floorCount: (j['floor_count'] ?? 0) as int,
    unitCount: (j['unit_count'] ?? 0) as int,
    builtYear: j['built_year'] as int?,
    managerPhone: j['manager_phone'] as String?,
    emergencyPhone: j['emergency_phone'] as String?,
    notes: j['notes'] as String?,
  );

  final String id;
  final String name;
  final String? address;
  final int blockCount;
  final int floorCount;
  final int unitCount;
  final int? builtYear;
  final String? managerPhone;
  final String? emergencyPhone;
  final String? notes;

  Map<String, dynamic> toJson() => {
    'name': name,
    if (address != null) 'address': address,
    'block_count': blockCount,
    'floor_count': floorCount,
    'unit_count': unitCount,
    if (builtYear != null) 'built_year': builtYear,
    if (managerPhone != null && managerPhone!.isNotEmpty)
      'manager_phone': managerPhone,
    if (emergencyPhone != null && emergencyPhone!.isNotEmpty)
      'emergency_phone': emergencyPhone,
    if (notes != null && notes!.isNotEmpty) 'notes': notes,
  };
}

/// Unit status enum values stay locale-neutral on the wire (server tokens);
/// Persian labels come from [unitStatusLabel].
const unitStatuses = ['active', 'vacant', 'occupied', 'inactive'];

class Unit {
  Unit({
    required this.id,
    required this.buildingId,
    required this.number,
    this.block,
    this.floor = 0,
    required this.areaM2,
    this.parkingCount = 0,
    this.parkingNumbers,
    this.storageCount = 0,
    this.storageNumbers,
    this.status = 'active',
    this.notes,
  });

  factory Unit.fromJson(Map<String, dynamic> j) => Unit(
    id: j['id'] as String,
    buildingId: j['building_id'] as String,
    number: (j['number'] ?? '') as String,
    block: j['block'] as String?,
    floor: (j['floor'] ?? 0) as int,
    areaM2: (j['area_m2'] ?? 0) as int,
    parkingCount: (j['parking_count'] ?? 0) as int,
    parkingNumbers: j['parking_numbers'] as String?,
    storageCount: (j['storage_count'] ?? 0) as int,
    storageNumbers: j['storage_numbers'] as String?,
    status: (j['status'] ?? 'active') as String,
    notes: j['notes'] as String?,
  );

  final String id;
  final String buildingId;
  final String number;
  final String? block;
  final int floor;
  final int areaM2;
  final int parkingCount;
  final String? parkingNumbers;
  final int storageCount;
  final String? storageNumbers;
  final String status;
  final String? notes;

  Map<String, dynamic> toJson() => {
    'number': number,
    if (block != null && block!.isNotEmpty) 'block': block,
    'floor': floor,
    'area_m2': areaM2,
    'parking_count': parkingCount,
    if (parkingNumbers != null && parkingNumbers!.isNotEmpty)
      'parking_numbers': parkingNumbers,
    'storage_count': storageCount,
    if (storageNumbers != null && storageNumbers!.isNotEmpty)
      'storage_numbers': storageNumbers,
    'status': status,
    if (notes != null && notes!.isNotEmpty) 'notes': notes,
  };
}

/// One audit entry of a unit's change history (`GET /units/{id}/history`).
class UnitHistoryEntry {
  factory UnitHistoryEntry.fromJson(Map<String, dynamic> j) =>
      UnitHistoryEntry(id: j['id'] as int, action: (j['action'] ?? '') as String);

  UnitHistoryEntry({required this.id, required this.action});

  final int id;
  final String action;
}
