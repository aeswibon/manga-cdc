-- +goose Up
-- +goose StatementBegin

-- Seed manga series
INSERT INTO manga_series (id, source_id, title, author, artist, description, cover_url, status, source_url, latest_chapter, last_checked, is_active)
VALUES
(
    'a1c3b275-c93f-4279-a17d-2b4742e47444',
    'md-1',
    'One Piece',
    'Eiichiro Oda',
    'Eiichiro Oda',
    'Gol D. Roger, a man referred to as the King of the Pirates, is set to be executed by the World Government...',
    'https://mangadex.org/covers/a1c3b275-c93f-4279-a17d-2b4742e47444/92330a10-2440-410a-8bf7-4632875f10b2.jpg',
    'ONGOING',
    'https://mangadex.org/title/a1c3b275-c93f-4279-a17d-2b4742e47444/one-piece',
    1115.00,
    NOW(),
    true
),
(
    '321e481a-641e-40d9-93b5-74c055272a5a',
    'md-2',
    'Solo Leveling',
    'Chugong',
    'DUBU (REDICE STUDIO)',
    'In a world where hunters must battle deadly monsters to protect mankind, Sung Jin-Woo, the weakest hunter...',
    'https://mangadex.org/covers/321e481a-641e-40d9-93b5-74c055272a5a/d32f418b-4b11-477d-bb62-43d92ccb7cd8.jpg',
    'COMPLETED',
    'https://mangadex.org/title/321e481a-641e-40d9-93b5-74c055272a5a/solo-leveling',
    200.00,
    NOW(),
    false
),
(
    '3331828f-7c15-46a1-a672-2d12e698889a',
    'as-1',
    'The Beginning After the End',
    'TurtleMe',
    'Fuyuki 23',
    'King Grey has unrivaled strength, wealth, and prestige in a world governed by martial ability. However...',
    'https://mangadex.org/covers/3331828f-7c15-46a1-a672-2d12e698889a/9903b412-2439-440a-91ff-2f63812d1b09.jpg',
    'ONGOING',
    'https://asuracomics.com/manga/the-beginning-after-the-end',
    185.00,
    NOW(),
    true
)
ON CONFLICT (source_id) DO NOTHING;

-- Seed chapters
INSERT INTO chapters (id, series_id, chapter_num, title, url, release_date, is_new)
VALUES
(
    'c1c3b275-c93f-4279-a17d-2b4742e47444',
    'a1c3b275-c93f-4279-a17d-2b4742e47444',
    1115.00,
    'The Message of Void',
    'https://mangadex.org/chapter/one-piece-1115',
    NOW() - INTERVAL '5 minutes',
    true
),
(
    'c21e481a-641e-40d9-93b5-74c055272a5a',
    '3331828f-7c15-46a1-a672-2d12e698889a',
    185.00,
    'Training Arc Commences',
    'https://asuracomics.com/chapter/tbate-185',
    NOW() - INTERVAL '12 minutes',
    true
),
(
    'c331828f-7c15-46a1-a672-2d12e698889a',
    '321e481a-641e-40d9-93b5-74c055272a5a',
    200.00,
    'Epilogue — The Eternal Monarch',
    'https://mangadex.org/chapter/solo-leveling-200',
    NOW() - INTERVAL '60 minutes',
    false
)
ON CONFLICT (series_id, chapter_num) DO NOTHING;

-- Seed notification logs
INSERT INTO notification_logs (id, chapter_id, status, channel, error_message, sent_at)
VALUES
(
    'd1c3b275-c93f-4279-a17d-2b4742e47444',
    'c1c3b275-c93f-4279-a17d-2b4742e47444',
    'SENT',
    'discord',
    NULL,
    NOW() - INTERVAL '5 minutes'
),
(
    'd21e481a-641e-40d9-93b5-74c055272a5a',
    'c21e481a-641e-40d9-93b5-74c055272a5a',
    'SENT',
    'telegram',
    NULL,
    NOW() - INTERVAL '12 minutes'
),
(
    'd331828f-7c15-46a1-a672-2d12e698889a',
    'c331828f-7c15-46a1-a672-2d12e698889a',
    'FAILED',
    'slack',
    'Webhook returned status 404 Not Found',
    NULL
)
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd
