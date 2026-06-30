package errors

var ErrAuthUserIDRequired = InvalidInput("session.auth_user_id_required", "В запросе отсутствует идентификатор пользователя")
