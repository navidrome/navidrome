# Test Plan: Endpoint `/rest/star` - Marcar Canción como Favorita

## Flujo del Endpoint

```
POST /rest/star?u=admin&t=TOKEN&s=SALT&f=json&v=1.8.0&c=NavidromeUI&id=MEDIA_FILE_ID
```

**Parámetros:**
- `u`: usuario (admin)
- `t`: token de sesión
- `s`: salt (para checksum)
- `f`: formato (json)
- `v`: versión API Subsonic
- `c`: cliente
- `id`: id del media_file a marcar como favorita

**Operación en BD:**
1. Verificar que el usuario `admin` existe en tabla `USER`
2. Verificar que el media_file con `id` existe
3. Insertar/actualizar en tabla `ANNOTATION`:
   - `user_id` = admin
   - `item_id` = MEDIA_FILE_ID
   - `item_type` = 'song'
   - `starred` = true
   - `starred_at` = current_timestamp

---

## Tests Propuestos (EVALUACIÓN)

### ✅ TEST 1: Verificar Foreign Key de ANNOTATION → USER
```sql
-- BIEN: Comprueba que NO puedes insertar user_id inválido
INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('INVALID_USER_ID', 'song123', 'song', true);
-- DEBE FALLAR con constraint error
```
**Evaluación:** ✅ BUENO - Es fundamental validar integridad referencial.

**Mejora sugerida:** También incluir ON DELETE CASCADE - prueba que al borrar un usuario, se borren sus anotaciones.

---

### ✅ TEST 2: Verificar Campos NOT NULL
```sql
-- BIEN: Campos obligatorios
INSERT INTO annotation (user_id, item_id, item_type) 
VALUES (NULL, 'song123', 'song');
-- DEBE FALLAR
```
**Evaluación:** ✅ BUENO - Pero incompleto. Falta probar múltiples campos.

**Mejora sugerida:** 
- Probar todos los campos NOT NULL (`user_id`, `item_id`, `item_type`, `starred`)
- Probar valores vacíos/default

---

### ✅ TEST 3: Insertar Canción + Annotation + SELECT con JOIN
```go
// Insertar media_file
INSERT INTO media_file (id, path, title) VALUES ('song123', '/path/song.mp3', 'My Song');

// Insertar annotation
INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin_id', 'song123', 'song', true);

// Verificar con JOIN
SELECT mf.title, ann.starred, ann.starred_at 
FROM media_file mf 
JOIN annotation ann ON mf.id = ann.item_id 
WHERE ann.user_id = 'admin_id' AND ann.item_type = 'song';
```
**Evaluación:** ✅ EXCELENTE - Valida el flujo completo.

---

### ✅ TEST 4: UPDATE - Marcar Favorita
```sql
UPDATE annotation 
SET starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123' AND item_type = 'song';
```
**Evaluación:** ✅ BUENO - Pero puede mejorase con transacciones y validaciones de timestamps.

---

## Tests Recomendados ADICIONALES

### 📌 TEST 5: Constraint UNIQUE de ANNOTATION
```sql
-- ANNOTATION tiene UNIQUE (user_id, item_id, item_type)
-- Intentar insertar dos veces el mismo registro debe fallar en la 2ª

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin_id', 'song123', 'song', true);

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin_id', 'song123', 'song', false);
-- DEBE FALLAR con UNIQUE constraint error
```

### 📌 TEST 6: Verificar Polimorfismo de ANNOTATION
```sql
-- item_type puede ser 'song', 'album', 'artist', etc.
INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin_id', 'album456', 'album', true);

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin_id', 'artist789', 'artist', true);

-- Verificar que item_id NO existe en correspondiente tabla
-- Esto validaría que sin FK explícitas, hay riesgo de datos huérfanos
```

### 📌 TEST 7: Timestamp de starred_at
```sql
-- Verificar que starred_at se actualiza al cambiar starred
UPDATE annotation 
SET starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123';

-- Esperar 1 segundo
-- UPDATE nuevamente
UPDATE annotation 
SET starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123';

-- El timestamp debe ser más reciente
SELECT starred_at FROM annotation 
WHERE user_id = 'admin_id' AND item_id = 'song123';
```

### 📌 TEST 8: play_count vs starred
```sql
-- Verificar que play_count se incrementa independientemente de starred
INSERT INTO annotation (user_id, item_id, item_type, starred, play_count) 
VALUES ('admin_id', 'song123', 'song', false, 0);

UPDATE annotation SET play_count = play_count + 1 
WHERE user_id = 'admin_id' AND item_id = 'song123';

-- Marcar como favorita
UPDATE annotation SET starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123';

-- Verificar que ambos campos persisten
SELECT play_count, starred FROM annotation 
WHERE user_id = 'admin_id' AND item_id = 'song123';
```

### 📌 TEST 9: ON DELETE CASCADE - User Deletion
```sql
-- Insertar usuario con annotations
INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at) 
VALUES ('test_user', 'testuser', 'hashedpwd', false, NOW(), NOW());

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('test_user', 'song123', 'song', true);

-- Verificar que existe
SELECT COUNT(*) FROM annotation WHERE user_id = 'test_user';
-- Debe retornar 1

-- Borrar usuario
DELETE FROM user WHERE id = 'test_user';

-- Verificar que las anotaciones se borraron en cascada
SELECT COUNT(*) FROM annotation WHERE user_id = 'test_user';
-- Debe retornar 0 (borración en cascada funcionó)
```

### 📌 TEST 10: Integridad con Multiple Users
```sql
-- Verificar que cada usuario tiene anotaciones independientes
INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at) 
VALUES ('admin', 'admin', 'pwd1', true, NOW(), NOW());

INSERT INTO user (id, user_name, password, is_admin, created_at, updated_at) 
VALUES ('user2', 'user2', 'pwd2', false, NOW(), NOW());

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin', 'song123', 'song', true);

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('user2', 'song123', 'song', false);

-- Verificar que cada usuario ve su estado
SELECT starred FROM annotation 
WHERE user_id = 'admin' AND item_id = 'song123';
-- Retorna true

SELECT starred FROM annotation 
WHERE user_id = 'user2' AND item_id = 'song123';
-- Retorna false
```

### 📌 TEST 11: Rating + Starred Independientes
```sql
-- Verificar que puedo calificar y marcar favorita independientemente
UPDATE annotation 
SET rating = 5, starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123';

-- Cambiar solo el rating
UPDATE annotation 
SET rating = 3 
WHERE user_id = 'admin_id' AND item_id = 'song123';

-- Verificar que starred se mantiene
SELECT rating, starred FROM annotation 
WHERE user_id = 'admin_id' AND item_id = 'song123';
-- Debe retornar: rating=3, starred=true
```

### 📌 TEST 12: NULL en Campos Opcionales
```sql
-- Algunos campos pueden ser NULL (play_date, rating, etc.)
INSERT INTO annotation (user_id, item_id, item_type, starred, play_count, rating, play_date) 
VALUES ('admin_id', 'song456', 'song', true, NULL, NULL, NULL);

SELECT * FROM annotation 
WHERE user_id = 'admin_id' AND item_id = 'song456';
-- Debe permitir NULLs en play_count, rating, play_date
```

### 📌 TEST 13: Transacción - Rollback si Falla
```sql
BEGIN TRANSACTION;

INSERT INTO media_file (id, path, title) VALUES ('song_new', '/path/new.mp3', 'New Song');

INSERT INTO annotation (user_id, item_id, item_type, starred) 
VALUES ('admin_id', 'song_new', 'song', true);

-- Simular error
-- ROLLBACK;

-- Verificar que nada se insertó
SELECT COUNT(*) FROM media_file WHERE id = 'song_new';
SELECT COUNT(*) FROM annotation WHERE item_id = 'song_new';
-- Ambos deben retornar 0
```

### 📌 TEST 14: Índices - Performance
```sql
-- Verificar que existen índices en:
-- 1. user_id (para búsquedas rápidas)
-- 2. (user_id, item_id, item_type) composite

EXPLAIN QUERY PLAN
SELECT * FROM annotation 
WHERE user_id = 'admin_id' AND item_id = 'song123' AND item_type = 'song';
-- Debe usar índice (no SCAN)
```

### 📌 TEST 15: Idempotencia - Star/Unstar Múltiples Veces
```sql
-- Marcar como favorita
UPDATE annotation 
SET starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123' AND item_type = 'song';

-- Marcar nuevamente (debe ser safe)
UPDATE annotation 
SET starred = true, starred_at = CURRENT_TIMESTAMP 
WHERE user_id = 'admin_id' AND item_id = 'song123' AND item_type = 'song';

-- Si record no existe, hacer INSERT (upsert pattern)
INSERT OR REPLACE INTO annotation (user_id, item_id, item_type, starred, starred_at)
VALUES ('admin_id', 'song123', 'song', true, CURRENT_TIMESTAMP);

-- Debe ser seguro (no duplicados, no errores)
```

---

## Matriz de Cobertura

| Test | Categoría | Importancia | Estado |
|------|-----------|-------------|--------|
| 1 | Foreign Key | ⭐⭐⭐⭐⭐ | ✅ Propuesto |
| 2 | Constraints | ⭐⭐⭐⭐⭐ | ✅ Propuesto |
| 3 | Flujo Completo | ⭐⭐⭐⭐⭐ | ✅ Propuesto |
| 4 | UPDATE | ⭐⭐⭐⭐⭐ | ✅ Propuesto |
| 5 | UNIQUE | ⭐⭐⭐⭐⭐ | 📌 Recomendado |
| 6 | Polimorfismo | ⭐⭐⭐⭐ | 📌 Recomendado |
| 7 | Timestamps | ⭐⭐⭐⭐ | 📌 Recomendado |
| 8 | Independencia datos | ⭐⭐⭐⭐ | 📌 Recomendado |
| 9 | Cascada | ⭐⭐⭐⭐⭐ | 📌 Recomendado |
| 10 | Multi-usuario | ⭐⭐⭐⭐ | 📌 Recomendado |
| 11 | Campos independientes | ⭐⭐⭐⭐ | 📌 Recomendado |
| 12 | NULLs opcionales | ⭐⭐⭐ | 📌 Recomendado |
| 13 | Transacciones | ⭐⭐⭐⭐ | 📌 Recomendado |
| 14 | Performance | ⭐⭐⭐⭐ | 📌 Recomendado |
| 15 | Idempotencia | ⭐⭐⭐⭐⭐ | 📌 Recomendado |

---

## Recomendación Final

**Tests CRÍTICOS (HACER PRIMERO):**
1. Foreign Key de ANNOTATION → USER (con cascada)
2. NOT NULL constraints
3. Flujo completo (INSERT + JOIN)
4. UPDATE starred flag
5. UNIQUE constraint (user_id, item_id, item_type)
6. Idempotencia (upsert safe)

**Tests IMPORTANTES (hacer después):**
7. Multi-usuario (aislamiento)
8. Timestamps accuracy
9. ON DELETE CASCADE
10. Performance con índices

**Tests COMPLEMENTARIOS (nice-to-have):**
11. Polimorfismo (item_type)
12. play_count independiente
13. Transacciones y rollback
14. NULL fields opcionales
