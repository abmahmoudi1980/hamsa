/// Role recorded on `users.role` (migration 0001; `superadmin` added by
/// migration 0010). Authorization itself is resolved server-side per
/// request (research.md R8) — this only drives client-side navigation.
enum UserRole { manager, resident, superadmin }

UserRole userRoleFromWire(String value) => switch (value) {
  'resident' => UserRole.resident,
  'superadmin' => UserRole.superadmin,
  _ => UserRole.manager,
};

String userRoleToWire(UserRole role) => switch (role) {
  UserRole.resident => 'resident',
  UserRole.superadmin => 'superadmin',
  UserRole.manager => 'manager',
};

/// The logged-in identity returned by `POST /auth/verify` (`user` object —
/// contracts/api.md).
class UserSession {
  const UserSession({
    required this.id,
    required this.name,
    required this.role,
    this.primaryBuildingId,
  });

  factory UserSession.fromJson(Map<String, dynamic> json) => UserSession(
    id: json['id']?.toString() ?? '',
    name: json['name']?.toString() ?? '',
    role: userRoleFromWire(json['role']?.toString() ?? ''),
    primaryBuildingId: json['primary_building_id']?.toString(),
  );

  final String id;
  final String name;
  final UserRole role;

  /// Present for managers; null for residents.
  final String? primaryBuildingId;

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'role': userRoleToWire(role),
    if (primaryBuildingId != null) 'primary_building_id': primaryBuildingId,
  };
}
