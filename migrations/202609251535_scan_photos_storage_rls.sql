-- Storage RLS for the scan-photos bucket: each user may only read/write/
-- delete objects under their own user_id/ prefix. Object paths are expected
-- to look like "<user_id>/<filename>".

create policy "scan_photos_select_own"
    on storage.objects for select
    using (bucket_id = 'scan-photos' and (storage.foldername(name))[1] = auth.uid()::text);

create policy "scan_photos_insert_own"
    on storage.objects for insert
    with check (bucket_id = 'scan-photos' and (storage.foldername(name))[1] = auth.uid()::text);

create policy "scan_photos_update_own"
    on storage.objects for update
    using (bucket_id = 'scan-photos' and (storage.foldername(name))[1] = auth.uid()::text)
    with check (bucket_id = 'scan-photos' and (storage.foldername(name))[1] = auth.uid()::text);

create policy "scan_photos_delete_own"
    on storage.objects for delete
    using (bucket_id = 'scan-photos' and (storage.foldername(name))[1] = auth.uid()::text);
