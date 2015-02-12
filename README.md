# ec2backup [![Build Status](https://travis-ci.org/marcy-go/ec2backup.svg?branch=master)](https://travis-ci.org/marcy-go/ec2backup)

Amazon EC2 Backup Command.

This command requires the IAM Role like:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateImage",
        "ec2:CreateSnapshot",
        "ec2:CreateTags",
        "ec2:DeleteSnapshot",
        "ec2:DeregisterImage",
        "ec2:DescribeImages",
        "ec2:DescribeInstances",
        "ec2:RegisterImage"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
```

## Usage

First, your instance(s) is required the tag that has the key named `Backup-Generation` and the number value of generations.

### Backup myself
```sh
/path/to/ec2backup self
```

### Backup all instanses(same region only)
```sh
/path/to/ec2backup all
```

# Changelog

See [CHANGELOG](https://github.com/marcy-go/ec2backup/blob/master/CHANGELOG.md)

## Contributing

1. Fork it ( https://github.com/marcy-go/ec2backup/fork )
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request

# License

[MIT License](https://github.com/marcy-go/ec2backup/blob/master/LICENSE.txt)
