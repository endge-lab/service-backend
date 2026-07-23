# Reference: наблюдаемый `Converter.Create`

```go
func (c *Converter) Create(ctx context.Context, input CreateConverterInput) (result *ConverterWithFolder, err error) {
	const op = "converter.create"
	ctx, cancel := context.WithTimeout(ctx, converterOperationTimeout)
	defer cancel()

	ctx, observed := c.observer.Start(ctx, op, nil, nil)
	defer observed.End(&err)

	if err = normalizeAndValidateCreateInput(&input); err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(
		"converter.create.input_validated",
		"converter create input validated",
		[]attribute.KeyValue{
			attribute.String("project.identity", input.ProjectIdentity),
			attribute.String("converter.identity", input.Identity),
		},
		zap.String("project_identity", input.ProjectIdentity),
		zap.String("converter_identity", input.Identity),
	)

	project, err := c.projectRepository.GetByIdentity(ctx, input.ProjectIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("project_identity", input.ProjectIdentity))
		return nil, err
	}
	observed.RecordStep(
		"converter.create.project_resolved",
		"project resolved for converter create",
		[]attribute.KeyValue{attribute.String("project.id", project.ID.String())},
		zap.String("project_id", project.ID.String()),
	)

	folder, err := c.resolveFolder(ctx, project.ID, input.FolderIdentity)
	if err != nil {
		logOperationError(observed.Logger(), op, err, zap.String("folder_identity", input.FolderIdentity))
		return nil, err
	}
	observed.RecordStep(
		"converter.create.folder_resolved",
		"folder resolved for converter create",
		[]attribute.KeyValue{attribute.String("folder.id", folder.ID.String())},
		zap.String("folder_id", folder.ID.String()),
	)

	exists, err := c.converterRepository.ExistsByIdentity(ctx, project.ID, input.Identity)
	if err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	if exists {
		err = apperrors.Conflict("identity_conflict", "converter identity already exists")
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(
		"converter.create.identity_available",
		"converter identity availability confirmed",
		[]attribute.KeyValue{attribute.String("converter.identity", input.Identity)},
		zap.String("converter_identity", input.Identity),
	)

	converter, err := c.converterRepository.Create(ctx, newConverter(project, folder, input))
	if err != nil {
		logOperationError(observed.Logger(), op, err)
		return nil, err
	}
	observed.RecordStep(
		"converter.create.persisted",
		"converter persisted",
		[]attribute.KeyValue{attribute.String("converter.id", converter.ID.String())},
		zap.String("converter_id", converter.ID.String()),
	)

	return converterWithFolder(converter, folder.Identity), nil
}
```

Все attributes должны описывать завершённый шаг, а не становиться labels
метрик. При ошибке success event не пишется: `logOperationError` и
`observed.End(&err)` уже фиксируют failure-path.
