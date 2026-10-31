ALTER TABLE "release"
    ADD COLUMN freeleech BOOLEAN DEFAULT FALSE;

ALTER TABLE "release"
    ADD COLUMN freeleech_percent INTEGER DEFAULT 0;
