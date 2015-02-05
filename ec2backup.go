package main

import (
  "fmt"
  "os"
  "strings"
  "strconv"
  "net/http"
  "io/ioutil"
  "github.com/awslabs/aws-sdk-go/aws"
  "github.com/awslabs/aws-sdk-go/gen/ec2"
  "github.com/mitchellh/cli"
)

type Own struct{}

func (s *Own) Help() string {
  return "ec2bk own"
}

func (s *Own) Synopsis() string {
  return "Backing up myself."
}

func (s *Own) Run() int {
  ins_id, err := getInstanceId()
  if err != nil {
    return 1, err
  }
  err := backup([]string{ins_id})
  if err != nil {
    return 1, err
  }
  return 0, nil
}

type All struct{}

func (s *All) Help() string {
  return "ec2bk all"
}

func (s *All) Synopsis() string {
  return "Backing up all instances."
}

func (s *All) Run() int {
  err := backup([]string)
  if err != nil {
    return 1, err
  }
  return 0, nil
}

func backup(ids []string) error {
  region, err := getRegion()
  if err != nil {
    return 1, err
  }
  req := ec2.DescribeInstancesRequest{}
  req.DescribeInstancesRequest = ids
  ec2_cli := ec2.New(aws.IAMCreds(), region, nil)
  res, err := ec2_cli.DescribeInstances(req)
  if err != nil {
    return err
  }

  for _, r := range res.Reservations {
    for _, ins := range r.Instances {
      name := ""
      gen  := 0
      for _, tag := range ins.Tags {
        key := strings.Trim(" ", tag.Key)
        val := strings.Trim(" ", tag.Value)
        if key == "Name" {
          name := tag.Value
        }
        if val == "Backup-Generation" {
          gen, err := strconv.Atoi(val)
          if err != nil {
            return err
          }
        }
      }
      if gen > 0 {
        var ci_req ec2.CreateImageRequest
        ci_req.BlockDeviceMappings = ins.BlockDeviceMappings
        ci_req.InstanceID          = ins.InstanceID
        ci_req.Name                = name + time.Now().Format("200601021504")
        ci_req.NoReboot            = true
        ci_ret, err := ec2_cli.CreateImage(ci_req)
        if err != nil {
          return err
        }
      }

    }
  }
  return nil
}

func getMetaData(url string) (string, error) {
  res, err := http.Get("http://169.254.169.254/latest/meta-data" + url)
  defer res.Body.Close()
  if err != nil {
    return nil, err
  }
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    return nil, err
  }
  return string(body), nil
}

func getInstanceId() (string, error) {
  return getMetaData("/instance-id")
}

func getRegion() (string, error) {
  meta, err := getMetaData("/placement/availability-zone")
  if err != nil {
    return nil, err
  }
  end  := len(meta) - 1
  return meta[0:end], nil
}

func main() {
  c := cli.NewCLI("ec2backup", "0.1.0")
  c.Args = os.Args[1:]
  c.Commands = map[string]cli.CommandFactory{
    "own": func() (cli.Command, error) {
      return &Own{}, nil
    },
    "all": func() (cli.Command, error) {
      return &All{}, nil
    },
  }
  ret, err := c.Run()
  if err != nil {
    fmt.Fprintf(os.Stderr, err.Error())
  }
  os.Exit(ret)
}
