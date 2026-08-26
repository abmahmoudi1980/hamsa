/// Role recorded on `users.role` (migration 0001). Authorization itself is
/// resolved server-side per request (research.md R8) — this only drives
/// client-side navigation.
enum UserRole { manager, resident }

UserRole userRoleFromWire(String value) =>
    value == 'resident' ? UserRole.resident : UserRole.manager;

String userRoleToWire(UserRole role) =>
    role == UserRole.resident ? 'resident' : 'manager';

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
